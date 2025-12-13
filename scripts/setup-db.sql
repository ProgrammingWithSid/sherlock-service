-- Database setup SQL script for Code-Sherlock
-- Run this script to create the database and enable required extensions

-- Create database (run this as postgres superuser)
-- CREATE DATABASE sherlock;

-- Connect to the sherlock database and run the following:

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- Or if uuid-ossp is not available, use pgcrypto
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Note: Tables will be created automatically by the application migrations
-- when you start the server. This script just sets up the database.


