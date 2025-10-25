# AsyncTask - 通用异步任务处理模块

## Requirements
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://golang.org/)
[![GoFrame](https://img.shields.io/badge/GoFrame-v2.7-green)](https://goframe.org/)

一个基于 GoFrame 的通用异步任务处理模块，支持任务调度、自动重试、超时恢复等功能。可以轻松集成到任何 Go 项目中。

## ✨ Feature

- 🚀 **开箱即用**：自动创建数据表，无需手动执行 SQL
- 🔒 **并发安全**：乐观锁机制，防止并发冲突
- 🔄 **智能重试**：指数退避重试策略，自动处理失败任务
- 📊 **超时监控**：任务处理超市超时，自动重置
- ⏰ **定时任务**：支持定时任务
- 🔧 **灵活配置**：支持自定义表名、数据库、重试策略等
- 🎯 **类型安全**：使用方自定义任务类型

## ⚙️ 配置说明

```go
type Config struct {
    // 数据库DSN（必填）
    DSN string
    
    // 数据库名称（必填）
    Database string
    
    // 数据库分组名，默认 "default"
    Group string
    
    // 表名，默认 "t_async_task"
    TableName string
    
    // 日志接口，默认使用标准log
    Logger Logger
    
    // 工作线程初始化间隔，默认10秒
    InitInterval time.Duration
    
    // 查询间隔（无任务时），默认30秒
    QueryInterval time.Duration
    
    // 错误休眠间隔，默认3秒
    ErrSleepInterval time.Duration
    
    // 超时检查间隔，默认24小时
    TimeoutCheckInterval time.Duration
    
    // 任务超时时长，默认24小时
    TaskTimeout time.Duration
    
    // 退避重试间隔列表
    BackoffIntervals []time.Duration
}
```

## 数据表设计
```sql
CREATE TABLE IF NOT EXISTS `t_async_task` (
  `id` BIGINT(20) AUTO_INCREMENT NOT NULL COMMENT '主键ID',
  `custom_id` VARCHAR(40) DEFAULT '' COMMENT '自定义任务ID',
  `task_type` TINYINT(1) NOT NULL COMMENT '任务类型',
  `status` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '任务状态(0:Pending, 1:Processing, 2:Success)',
  `content` TEXT NOT NULL COMMENT '任务执行参数',
  `retry_count` INT(11) NOT NULL DEFAULT 0 COMMENT '重试次数',  
  `next_retry_time` BIGINT(20) NOT NULL COMMENT '下次处理时间(默认等于创建时间)',
  `version` INT(11) NOT NULL DEFAULT 0 COMMENT '版本标识',  
  `create_time` BIGINT(20) NOT NULL COMMENT '创建时间',
  `update_time` BIGINT(20) NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_custom_id` (`custom_id`),
  KEY `idx_type_status_time` (`task_type`, `status`, `next_retry_time`),
  KEY `idx_status_next_retry_time` (`status`, `next_retry_time`),
  KEY `idx_status_update_time` (`status`, `update_time`)
) ENGINE=InnoDB COMMENT='异步任务表';

CREATE TABLE IF NOT EXISTS `t_async_task_history` (
    `id` BIGINT(20) AUTO_INCREMENT NOT NULL COMMENT '主键ID',
    `task_id` BIGINT(20) NOT NULL COMMENT '任务ID',
    `round` INT(11) NOT NULL COMMENT '第几次执行',
    `status` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '任务是否执行成功(0:失败, 1:成功)',
    `result` TEXT COMMENT '任务执行结果',
    `start_time` BIGINT(20) NOT NULL COMMENT '任务执行开始时间',
    `end_time` BIGINT(20) NOT NULL COMMENT '任务执行结束时间',
    `duration` BIGINT(20) NOT NULL COMMENT '任务执行时间间隔',
    PRIMARY KEY (`id`),
    KEY `idx_task_id` (`task_id`)
) ENGINE=InnoDB COMMENT='任务执行记录表';
```

# TODO List 
- 创建AsyncTaskManager时，支持自定义 maxRetry，作用于 所有工作队列。如果不指定，使用现有回退机制，不限制重试次数。
- 支持指定单个任务的最大重试次数及重试策略？增加死信类型，如果超过最大重试次数，置为 死信，后续不再处理。




## 📦 安装

```bash
go get github.com/yyboo/asynctask
```

## 🎯 使用场景

- ✅ 异步发送邮件/短信
- ✅ 延迟任务处理
- ✅ 定时任务调度
- ✅ 数据批量处理
- ✅ 第三方API调用（需要重试）
- ✅ 事件驱动架构

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
