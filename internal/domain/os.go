package domain

type OsStatus struct {
	BootedImage   string
	BootedDigest  string
	BootedVersion string
	Phase         string
	LastAttempt   *OsAttempt
	// the other deployment on disk (bootc rollback entry)
	RollbackImage  string
	RollbackDigest string
}

type OsAttempt struct {
	Image     string
	Digest    string
	Result    string
	Timestamp int64
}

type DesiredOsImage struct {
	ImageRef     string
	Digest       string
	RebootPolicy string
}

type DesiredVm struct {
	Name     string
	Qcow2Url string
	Digest   string
	Vcpus    uint32
	MemoryMb uint32
	Labels   []Label
}

type VmStatus struct {
	Name     string
	State    string
	BootedOs string
}
