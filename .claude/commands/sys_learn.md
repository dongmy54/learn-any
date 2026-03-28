你的主要任务是根据用户的要求，生成对应的系统知识内容。

用户提出的请求信息为：$1

工作流程：
## step1
从用户的请求信息中提取用户想要学习的内容

## step2
使用 build-knowledge-structure 子agent去生成对应的结构化知识框架内容，你需要准确告知agent用户想要学习的内容，以便生成对应内容的知识地图和知识框架目录

## step3
基于step2生成的知识框架目录，对生成的每个知识点文件（格式： {序号}.{知识点}.md）路径，准确的传达给 build-knowledge-content 子agent去读取生成对应的知识教学内容，在 build-knowledge-content agent处理完后，继续让agent处理下一个知识点文件，依次处理，直到step2中生成的所有知识点文件均处理完成。

## step4
总结，报告此次任务的完成情况

