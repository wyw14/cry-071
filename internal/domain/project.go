package domain

type ProjectMetadata struct {
	Name             string `json:"name"`
	RequirementCode  string `json:"requirement_code"`
	GlobalSequence   int    `json:"global_sequence"`
	OriginalCategory string `json:"original_category"`
	SourceReference  string `json:"source_reference"`
}

func Metadata() ProjectMetadata {
	return ProjectMetadata{
		Name: "公共空间反馈处理追踪模块", RequirementCode: "GO-FE-021", GlobalSequence: 71,
		OriginalCategory: "Feature迭代", SourceReference: "word2.xlsx / Sheet1 / 第 751 行",
	}
}
