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
	InstanceID string
	State      string
	PublicIP   string
}

func (i Info) ID() string {
	return i.InstanceID
}

func (i Info) Status() string {
	return i.State
}
