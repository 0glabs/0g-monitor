-- Create the database
CREATE DATABASE IF NOT EXISTS storage_service;

-- Use the database
USE storage_service;

-- Create the table with two fields: discord_id and status
CREATE TABLE IF NOT EXISTS user_status (
    discord_id VARCHAR(255) NOT NULL,
    status ENUM('CONNECTED', 'DISCONNECTED') NOT NULL,
    PRIMARY KEY (discord_id)
);
