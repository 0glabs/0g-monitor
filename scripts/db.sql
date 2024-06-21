-- Create the database
CREATE DATABASE IF NOT EXISTS user_service;

-- Use the database
USE user_service;

-- Create the table for the user storage status
CREATE TABLE IF NOT EXISTS user_storage_status (
    ip VARCHAR(255) NOT NULL,
    discord_id VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    status ENUM('CONNECTED', 'DISCONNECTED') NOT NULL,
    PRIMARY KEY (ip)
);
