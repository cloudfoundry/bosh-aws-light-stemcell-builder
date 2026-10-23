package driver

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"light-stemcell-builder/config"
	"light-stemcell-builder/resources"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestSnapshotVisibility(t *testing.T) {
	for _, copied := range []bool{false, true} {
		for _, visibility := range []string{resources.PrivateAmiAccessibility, resources.PublicAmiAccessibility, ""} {
			name := visibility
			if copied {
				name += "-copy"
			}
			t.Run(name, func(t *testing.T) {
				var permissions []url.Values
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if err := r.ParseForm(); err != nil {
						t.Error(err)
						http.Error(w, "bad request", 400)
						return
					}
					action := r.Form.Get("Action")
					w.Header().Set("Content-Type", "text/xml")
					write := func(body string) {
						if _, err := io.WriteString(w, body); err != nil {
							t.Error(err)
						}
					}
					switch action {
					case "CreateSnapshot":
						write(`<CreateSnapshotResponse><snapshotId>snap-test</snapshotId></CreateSnapshotResponse>`)
					case "CopyImage":
						write(`<CopyImageResponse><imageId>ami-test</imageId></CopyImageResponse>`)
					case "DescribeSnapshots":
						write(`<DescribeSnapshotsResponse><snapshotSet><item><snapshotId>snap-test</snapshotId><status>completed</status></item></snapshotSet></DescribeSnapshotsResponse>`)
					case "DescribeImages":
						write(`<DescribeImagesResponse><imagesSet><item><imageId>ami-test</imageId><imageState>available</imageState><rootDeviceName>/dev/sda1</rootDeviceName><blockDeviceMapping><item><deviceName>/dev/sda1</deviceName><ebs><snapshotId>snap-test</snapshotId></ebs></item></blockDeviceMapping></item></imagesSet></DescribeImagesResponse>`)
					case "ModifySnapshotAttribute":
						permissions = append(permissions, r.Form)
						write(`<ModifySnapshotAttributeResponse><return>true</return></ModifySnapshotAttributeResponse>`)
					case "CreateTags", "ModifyImageAttribute":
						write("<" + action + "Response><return>true</return></" + action + "Response>")
					default:
						t.Errorf("unexpected operation %s", action)
						http.Error(w, "unexpected", 400)
					}
				}))
				defer server.Close()
				client := ec2.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}, func(o *ec2.Options) { o.BaseEndpoint = aws.String(server.URL) })
				properties := resources.AmiProperties{Accessibility: visibility, SharedWithAccounts: []string{"123456789012"}}
				if copied {
					d := &SDKCopyAmiDriver{ec2Client: client, logger: log.New(io.Discard, "", 0), creds: config.Credentials{Region: "us-east-1"}}
					_, err := d.Create(resources.AmiDriverConfig{ExistingAmiID: "ami-source", DestinationRegion: "us-west-2", AmiProperties: properties})
					if err != nil {
						t.Fatal(err)
					}
				} else {
					d := &SDKSnapshotFromVolumeDriver{ec2Client: client, logger: log.New(io.Discard, "", 0)}
					_, err := d.Create(resources.SnapshotDriverConfig{VolumeID: "vol-test", AmiProperties: properties})
					if err != nil {
						t.Fatal(err)
					}
				}
				public := false
				shared := false
				for _, permission := range permissions {
					if permission.Get("UserGroup.1") == "all" {
						public = true
					}
					if permission.Get("UserId.1") == "123456789012" {
						shared = true
					}
				}
				if public != (visibility == resources.PublicAmiAccessibility) {
					t.Errorf("public=%v for visibility %q: %v", public, visibility, permissions)
				}
				if !shared {
					t.Errorf("configured account not shared: %v", permissions)
				}
			})
		}
	}
}
