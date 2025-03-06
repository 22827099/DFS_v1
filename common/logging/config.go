package logger

// 日志配置结构体
type Config struct {
	Level     string `yaml:"level"`     // 日志级别
	Format    string `yaml:"format"`    // json/text
	Output    string `yaml:"output"`    // console/file/both
	FilePath  string `yaml:"file_path"` // 文件路径
	MaxSizeMB int    `yaml:"max_size"`  // 单个文件最大MB
	KeepDays  int    `yaml:"keep_days"` // 保留天数
}
