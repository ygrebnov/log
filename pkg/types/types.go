package types

// Format controls handler output.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Kind string

const (
	KindStdOut Kind = "stdout"
	KindStdErr Kind = "stderr"
	KindFile   Kind = "file"
	KindRemote Kind = "remote"
)
