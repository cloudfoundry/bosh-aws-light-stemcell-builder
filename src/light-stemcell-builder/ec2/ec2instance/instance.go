package ec2instance

const (
	RunningStatus    = "running"
	PendingStatus    = "pending"
	TerminatedStatus = "terminated"
)

type Config struct {
	AmiID             string
	InstanceType      string
	AssociatePublicIP bool
	Region            string
}

type Info struct {
	InstanceID string `key:"INSTANCE" position:"0"`
	State      string `key:"INSTANCE" position:"4"`
	PublicIP   string `key:"INSTANCE" position:"15"`
}

func (i Info) ID() string {
	return i.InstanceID
}

func (i Info) Status() string {
	return i.State
}
