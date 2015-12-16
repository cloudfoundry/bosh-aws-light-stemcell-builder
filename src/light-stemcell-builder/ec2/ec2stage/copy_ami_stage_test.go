package ec2stage_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"light-stemcell-builder/ec2"
	"light-stemcell-builder/ec2/ec2ami"
	"light-stemcell-builder/ec2/ec2stage"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CopyAmi", func() {
	logger := log.New(ioutil.Discard, "", log.LstdFlags)

	Describe("Run()", func() {
		noopUndoer := func(aws ec2.AWS, amiCollection *ec2ami.Collection) error {
			return nil
		}

		It("passes the ami info to the runner", func() {
			var called bool
			var info ec2ami.Info

			dummyRunner := func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
				called = true
				info = amiInfo
				return &ec2ami.Collection{}, nil
			}

			amiInfo := ec2ami.Info{AmiID: "some-unique-id"}
			s := ec2stage.NewCopyAmiStage(dummyRunner, noopUndoer, dummyAWS{}, []string{})
			s.Run(logger, amiInfo)
			Expect(called).To(BeTrue())
			Expect(info).To(Equal(amiInfo))
		})

		It("returns an ami collection", func() {
			var called bool

			dummyRunner := func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
				called = true

				dummyCollection := ec2ami.NewCollection()

				for i := range destinations {
					dummyCollection.Add(destinations[i], ec2ami.Info{AmiID: fmt.Sprintf("dummy-ami-id-%d", i)})
				}

				return dummyCollection, nil
			}

			regions := []string{"dummy-region-1", "dummy-region-2", "dummy-region-3"}
			s := ec2stage.NewCopyAmiStage(dummyRunner, noopUndoer, dummyAWS{}, regions)

			res, err := s.Run(logger, ec2ami.Info{})
			Expect(called).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			var amis *ec2ami.Collection
			Expect(res).To(BeAssignableToTypeOf(amis))

			amis = res.(*ec2ami.Collection)
			Expect(amis.Get("dummy-region-1").AmiID).ToNot(BeEmpty())
			Expect(amis.Get("dummy-region-2").AmiID).ToNot(BeEmpty())
			Expect(amis.Get("dummy-region-3").AmiID).ToNot(BeEmpty())
		})

		It("returns errors from the runner", func() {
			errorRunner := func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
				return &ec2ami.Collection{}, errors.New("this is an error")
			}

			s := ec2stage.NewCopyAmiStage(errorRunner, noopUndoer, dummyAWS{}, []string{})
			_, err := s.Run(logger, ec2ami.Info{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("this is an error"))
		})

		It("expects input to be an ec2ami.Info", func() {
			dummyRunner := func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
				return &ec2ami.Collection{}, nil
			}

			s := ec2stage.NewCopyAmiStage(dummyRunner, noopUndoer, dummyAWS{}, []string{})
			_, err := s.Run(logger, 42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected type ec2ami.Info, got: int"))
		})
	})

	Describe("Rollback()", func() {
		It("passes the AMI collection to the undoer", func() {
			var called bool
			var collection *ec2ami.Collection

			dummyCollection := ec2ami.NewCollection()
			dummyCollection.Add("dummy-region-1", ec2ami.Info{AmiID: "dummy-ami-id-1"})

			dummyRunner := func(aws ec2.AWS, amiInfo ec2ami.Info, destinations []string) (*ec2ami.Collection, error) {
				return dummyCollection, nil
			}

			dummyUndoer := func(aws ec2.AWS, amiCollection *ec2ami.Collection) error {
				called = true
				collection = amiCollection
				return nil
			}

			s := ec2stage.NewCopyAmiStage(dummyRunner, dummyUndoer, dummyAWS{}, []string{})
			s.Run(logger, ec2ami.Info{})
			s.Rollback(logger)
			Expect(called).To(BeTrue())
			Expect(collection).To(Equal(dummyCollection))
		})
	})
})
