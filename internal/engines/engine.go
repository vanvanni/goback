package engines

const (
	EngineKindS3     = "s3"
	EngineKindDocker = "docker"
	EngineKindDump   = "dump"
)

type EngineKind string

func (k EngineKind) String() string {
	return string(k)
}

func (k EngineKind) IsStorageEngine() bool {
	return k == EngineKindS3
}

type Engine interface {
	Kind() EngineKind
}
