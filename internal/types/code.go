package types

type RepoEntry struct {
	Name string
	Path string
}

type CodeRef struct {
	RepoName   string
	FullPath   string
	FilePath   string
	LineSpec   string
	SourceReport string
}