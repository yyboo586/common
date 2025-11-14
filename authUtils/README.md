# 重构authUtils
- 记录颁发的每一个JWT令牌
- 后面令牌格式不局限于JWT格式，也可以是Opaque。

```sql
CREATE TABLE IF NOT EXISTS `t_token` (
    `id` VARCHAR(40) NOT NULL COMMENT '令牌唯一标识JTI',
    `user_id` VARCHAR(40) NOT NULL COMMENT '用户ID',
    `device_id` VARCHAR(40) NOT NULL DEFAULT '' COMMENT '设备ID',
    `content` TEXT COMMENT '自定义数据',
    `is_active` TINYINT(4) NOT NULL DEFAULT 1 COMMENT '令牌有效标识(0:无效,1:有效)',
    `create_time` BIGINT(20) NOT NULL COMMENT '令牌创建时间',
    `expire_time` BIGINT(20) NOT NULL COMMENT '令牌过期时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_device_id` (`device_id`)
)ENGINE = InnoDB COMMENT='令牌表';
```

## 模块说明
- `tokenUtils.Token` 在生成/刷新令牌时，自动向 `t_token` 写入记录（含 user/device 信息与自定义负载）。
- `Token.Parse` 会校验 `t_token` 中的 `is_active` 和 `expire_time`，确保被撤销或过期的令牌无法继续使用。
- `RevokeToken` / `RevokeDeviceToken` / `RevokeUserToken` 将匹配的记录标记为 `is_active=0`，并结合可选黑名单实现多维度封禁。

## 使用示例
```go
storeCfg := tokenUtils.DefaultTokenStoreConfig()
storeCfg.DSN = "mysql:..."

blacklistCfg := tokenUtils.DefaultBlacklistConfig()
blacklistCfg.DSN = "mysql:..."

tokenSrv := tokenUtils.NewToken(
    tokenUtils.WithTokenStoreConfig(storeCfg),
    tokenUtils.WithBlacklistConfig(blacklistCfg),
)
```