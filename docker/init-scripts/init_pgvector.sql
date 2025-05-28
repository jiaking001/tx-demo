-- 启用pgvector扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- 创建用户表（如果需要预先创建）
CREATE TABLE IF NOT EXISTS users (
     id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
     user_id varchar(36) NOT NULL UNIQUE,
    -- 添加用户名字段便于登录
     username varchar(100) NOT NULL UNIQUE,
     password varchar(255) NOT NULL,
    -- like 是关键字因此打上双引号
     "like" varchar(255),
     like_embedding vector(1024),
     created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
     updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- 添加软删除字段
     deleted_at timestamp NULL
);

-- 为deleted_at字段添加索引，提高查询性能
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
-- 为username字段添加索引，提高查询性能
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);