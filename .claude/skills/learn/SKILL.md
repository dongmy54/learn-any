---
name: learn
description: 为帮助用户学习知识/技能，通过调用多个学习相关的skill,为用户生成知识/技能相关的文档。
---


你的任务是，编排调用多个skill，为用户快速生成知识/技能相关的文档。
你不应该自己去生成知识/技能相关的文档，而是应该调用其他skill来生成。

## 工作流程：
1. 提取用户要学习的知识/技能
2. 编排调用skill,可以开启多agent来生成
  - 调用`quick-learn`skill,生成扫盲文章
  - 调用`beginner-tutorial`skill,生成上手教程
  - 调用`concept-extract`skill，生成概念提取





