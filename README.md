# Grant Matcherator

A modern grant matching platform built with Go and React, designed to connect grant providers with recipients based on compatibility and needs.

## Features

- **Smart Matching Algorithm**:
  - Matches grant providers with recipients based on sector alignment
  - Considers target groups and project stages
  - Factors in funding requirements and timelines
  - Real-time connection status updates

- **Real-time Features**:
  - Live chat between connected organizations
  - Online/offline status indicators
  - Unread message notifications
  - Typing indicators

## System Requirements

- Go 1.22 or higher
- PostgreSQL 13 or higher
- Node.js 18 or higher
- npm 8 or higher

## Installation and Setup

### Backend Setup

1. Create a database called "matcherator":
```bash
cd backend
psql -U postgres
CREATE DATABASE matcherator;
\q
```

2. Initialize the database schema:
```bash
cd backend
psql -U postgres -d matcherator -f init.sql
```

3. Run the backend server:
```bash
cd backend
go mod tidy
go run main.go
```

The backend server will start on http://localhost:3000

### Frontend Setup

1. Install dependencies and start the development server:
```bash
cd frontend
npm install
npm run dev
```

The frontend application will be available at http://localhost:8080

## Project Structure

### Backend
- `/handlers`: Request handlers for different features
  - `auth.go`: Authentication handlers
  - `chat.go`: Real-time chat functionality
  - `match.go`: Matching logic
  - `profile.go`: Profile management
  - `user.go`: User management
  - `status.go`: Online/offline status
  - `upload.go`: File upload handling

### Frontend
- `/src/components`: React components
- `/src/pages`: Page components
- `/src/constants`: Configuration constants
- `/src/lib`: Utility functions
- `/src/types`: TypeScript type definitions

## API Endpoints

### Authentication
- POST `/api/auth/signup`: Register new organization
- POST `/api/auth/login`: Organization login

### Profile
- GET `/api/me/profile`: Get current organization's profile
- PUT `/api/me/profile`: Update profile
- GET `/api/users/:id`: Get organization's basic info
- GET `/api/users/:id/profile`: Get organization's profile info
- GET `/api/users/:id/recipient-data`: Get recipient-specific data
- GET `/api/users/:id/provider-data`: Get provider-specific data

### Matching
- GET `/api/recommendations`: Get potential matches
- POST `/api/matches/:id/dismiss`: Dismiss a recommendation

### Connections
- POST `/api/connections`: Create a new connection
- GET `/api/connections`: Get current connections
- GET `/api/match-status/:id`: Check match status with another organization

### Chat
- WebSocket `/ws`: Real-time chat and status updates

## Database Configuration

The application uses the following database configuration:
```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=matcherator
DB_USER=postgres
DB_PASSWORD=postgres
```

## Development Notes

- The matching algorithm considers sector alignment, target groups, and project stages
- WebSocket connections handle real-time chat and status updates
- Profile pictures are stored as URLs in the database
- Authentication uses JWT tokens
- All API endpoints require authentication except signup and login
- The platform supports both grant providers and recipients with different data models

## Recent Updates
- Added profile pages for organizations with detailed information
- Implemented real-time WebSocket messaging with typing indicators
- Added filtering capabilities for potential matches by sector, target groups, and project stage
- Enhanced profile pages with organization-specific information
- Improved connection status management and real-time updates
- Added email display in profile pages
- Implemented proper authorization checks for user endpoints
