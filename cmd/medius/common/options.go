package common

type Options struct {
	AllowInsecureRegistry bool
	Registry              string
	DryRun                bool
	PublishImagesOptions  PublishImageOptions
	VerifyImagesOptions   VerifyImageOptions
	PublishDocsOptions    PublishDocsOptions
	Focus                 string
}

type PublishImageOptions struct {
	ForceBuild bool
	Workers    int
}

type VerifyImageOptions struct {
	Workers         int
	ClusterRegistry string
	Namespace       string
	Timeout         int
}

type PublishDocsOptions struct {
	TokenFile string
}
