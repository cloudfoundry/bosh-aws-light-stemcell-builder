package driver

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestImageURLsRetainSignaturesWithoutLoggingThem(t *testing.T) {
	var uploaded []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "13")
		} else {
			var err error
			if uploaded, err = io.ReadAll(r.Body); err != nil {
				t.Error(err)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("fixture-access", "fixture-secret", "")}, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	presigner := s3.NewPresignClient(client)
	imagePath := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(imagePath, []byte("fixture image"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, manifest := range []bool{false, true} {
		t.Run(map[bool]string{false: "image", true: "manifest"}[manifest], func(t *testing.T) {
			var logs bytes.Buffer
			cfg := resources.MachineImageDriverConfig{MachineImagePath: imagePath, BucketName: "fixture-bucket", VolumeSizeGB: 1, FileFormat: "RAW"}
			var image resources.MachineImage
			var err error
			if manifest {
				d := &SDKCreateMachineImageManifestDriver{s3Client: client, presignClient: presigner, logger: log.New(&logs, "", 0)}
				image, err = d.Create(cfg)
				if !strings.Contains(string(uploaded), "X-Amz-Signature") {
					t.Error("uploaded manifest lost signed URLs")
				}
			} else {
				d := &SDKCreateMachineImageDriver{s3Client: client, presignClient: presigner, logger: log.New(&logs, "", 0)}
				image, err = d.Create(cfg)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(image.DeleteURLs) == 0 {
				t.Fatal("missing cleanup URLs")
			}
			for _, raw := range image.DeleteURLs {
				if !strings.Contains(raw, "X-Amz-Signature") {
					t.Errorf("cleanup URL lost signature: %s", raw)
				}
			}
			if manifest && !strings.Contains(image.GetURL, "X-Amz-Signature") {
				t.Error("manifest URL lost signature")
			}
			if strings.Contains(logs.String(), "X-Amz-") || strings.Contains(logs.String(), "fixture-secret") {
				t.Fatalf("credentials logged: %s", logs.String())
			}
		})
	}
}

func TestDeleteURLSignaturesStayOutOfLogsAndErrors(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			received := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received = r.URL.Query().Get("X-Amz-Signature") == "fixture-signature"
				w.WriteHeader(status)
				if _, err := io.WriteString(w, "echoed fixture-signature "+r.URL.String()); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			var logs bytes.Buffer
			d := &SDKDeleteMachineImageDriver{logger: log.New(&logs, "", 0)}
			err := d.Delete(resources.MachineImage{DeleteURLs: []string{server.URL + "/image?X-Amz-Signature=fixture-signature"}})
			if !received {
				t.Fatal("DELETE request lost signature")
			}
			if (err == nil) != (status == http.StatusOK) {
				t.Fatalf("unexpected error: %v", err)
			}
			text := logs.String()
			if err != nil {
				text += err.Error()
			}
			if strings.Contains(text, "fixture-signature") || strings.Contains(text, "X-Amz-") {
				t.Fatalf("signed URL logged: %s", text)
			}
		})
	}
	server := httptest.NewServer(http.NotFoundHandler())
	raw := server.URL + "/image?X-Amz-Signature=fixture-signature"
	server.Close()
	var logs bytes.Buffer
	d := &SDKDeleteMachineImageDriver{logger: log.New(&logs, "", 0)}
	for _, rawURL := range []string{raw, "://invalid?X-Amz-Signature=fixture-signature"} {
		err := d.Delete(resources.MachineImage{DeleteURLs: []string{rawURL}})
		if err == nil {
			t.Fatal("expected request failure")
		}
		if strings.Contains(err.Error()+logs.String(), "fixture-signature") {
			t.Fatalf("failed request leaked signature: %v", err)
		}
	}
}

func TestManifestPresigningDoesNotLogCredentials(t *testing.T) {
	// Exercise generation independently so a local upload error cannot hide a logged URL.
	var logs bytes.Buffer
	client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("fixture-access", "fixture-secret", "")})
	d := &SDKCreateMachineImageManifestDriver{presignClient: s3.NewPresignClient(client), logger: log.New(&logs, "", 0)}
	_, err := d.generateManifest(context.Background(), "fixture-bucket", "image", 1, 1, "RAW")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "X-Amz-") {
		t.Fatal("signed manifest URLs logged")
	}
}

func TestVolumeManifestErrorsDoNotIncludeSignedURLs(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "text/xml")
					if _, err := io.WriteString(w, `<DescribeAvailabilityZonesResponse><availabilityZoneInfo><item><zoneName>test-zone</zoneName><zoneState>available</zoneState></item></availabilityZoneInfo></DescribeAvailabilityZonesResponse>`); err != nil {
						t.Error(err)
					}
					return
				}
				w.WriteHeader(status)
				if _, err := io.WriteString(w, `<manifest>https://example.invalid/image?X-Amz-Signature=fixture-signature<bad`); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client := ec2.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}, func(o *ec2.Options) { o.BaseEndpoint = aws.String(server.URL) })
			d := &SDKCreateVolumeDriver{ec2Client: client, logger: log.New(io.Discard, "", 0)}
			_, err := d.Create(resources.VolumeDriverConfig{MachineImageManifestURL: server.URL + "/manifest?X-Amz-Signature=fixture-signature"})
			if err == nil {
				t.Fatal("expected manifest error")
			}
			if strings.Contains(err.Error(), "fixture-signature") || strings.Contains(err.Error(), "X-Amz-") {
				t.Fatalf("manifest error leaked signed URL: %v", err)
			}
		})
	}
}
