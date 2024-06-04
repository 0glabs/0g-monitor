-- Create the database
CREATE DATABASE IF NOT EXISTS storage_service;

-- Use the database
USE storage_service;

-- Create the table for the user status
CREATE TABLE IF NOT EXISTS user_status (
    ip VARCHAR(255) NOT NULL,
    discord_id VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    status ENUM('CONNECTED', 'DISCONNECTED') NOT NULL,
    PRIMARY KEY (ip)
);
