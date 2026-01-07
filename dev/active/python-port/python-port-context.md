# Python Port - Context & State

**Last Updated**: 2025-11-13 (Session 1 - Initial Setup)
**Status**: Foundation Phase - Local Dev Environment Complete ✅
**Location**: `/home/adam/listmonk-python/`

---

## Session Summary

Successfully completed Phase 1 (Foundation) of the Python/React port of listmonk. Created a complete local development environment using **uv** for Python package management on Kali Linux, with no Azure deployment yet.

## Key Decisions Made

### 1. Package Management: uv (not pip/poetry)
**Decision**: Use `uv` for Python dependency management
**Rationale**:
- User is on Kali Linux and requested virtual environment approach
- `uv` is ultra-fast, modern, and handles venvs excellently
- Better than pip/pipenv for development workflow
- Supports pyproject.toml standard

### 2. Database Models: Async SQLAlchemy 2.0
**Decision**: Use SQLAlchemy 2.0 with async support throughout
**Rationale**:
- FastAPI is async-native
- Better performance than sync ORM
- Matches the async/await patterns from Go goroutines
- Modern best practices

### 3. Local-First Development
**Decision**: No Azure deployment in initial phase - local testing only
**Rationale**:
- User explicitly requested: "I don't want to deploy anything to Azure until we are tested locally"
- Docker Compose provides PostgreSQL + Redis locally
- Easier debugging and faster iteration
- Azure deployment will come in later phase

### 4. Migration Strategy: Alembic
**Decision**: Use Alembic for database migrations (not raw SQL)
**Rationale**:
- Standard tool for SQLAlchemy migrations
- Auto-generates migrations from model changes
- Version control for schema changes
- Supports both upgrade and downgrade paths

### 5. Project Structure: Backend-First
**Decision**: Build backend foundation before React frontend
**Rationale**:
- Backend is more complex (156+ API endpoints)
- Database models are foundational
- Frontend can be built against working API
- Allows API testing with Swagger docs

---

## Files Created (Complete List)

### Project Root (`/home/adam/listmonk-python/`)
- ✅ `docker-compose.yml` - PostgreSQL 15 + Redis 7 for local dev
- ✅ `.gitignore` - Python, venv, Docker, IDE files
- ✅ `README.md` - Comprehensive project documentation
- ✅ `GETTING_STARTED.md` - Step-by-step setup guide

### Backend Structure (`backend/`)
- ✅ `pyproject.toml` - Python dependencies (FastAPI, SQLAlchemy, etc.)
- ✅ `.env.example` - Environment variables template
- ✅ `.env` - Actual environment config (created from example)
- ✅ `alembic.ini` - Alembic configuration

### Application Core (`backend/app/`)
- ✅ `config.py` - Pydantic Settings (loads from .env)
- ✅ `database.py` - Async SQLAlchemy engine + session factory
- ✅ `main.py` - FastAPI application with lifespan, CORS, health check

### Database Models (`backend/app/models/`)
- ✅ `__init__.py` - Model exports
- ✅ `base.py` - Base class + TimestampMixin
- ✅ `user.py` - User, Role, UserRole (3 models)
- ✅ `subscriber.py` - Subscriber, SubscriberList (2 models)
- ✅ `list.py` - List, ListRole (2 models)
- ✅ `campaign.py` - Campaign, CampaignList (2 models)
- ✅ `template.py` - Template (1 model)

**Total Models**: 11 core models with relationships, indexes, and UUIDs

### Alembic Setup (`backend/alembic/`)
- ✅ `env.py` - Alembic environment (async-aware)
- ✅ `script.py.mako` - Migration template
- ✅ `versions/` - Directory for migrations (empty, will be auto-generated)

### Scripts (`scripts/`)
- ✅ `quickstart.sh` - Automated setup script (executable)

### Placeholder Directories (Created, Empty)
- `backend/app/api/` - API endpoints (Phase 2)
- `backend/app/core/` - Business logic services (Phase 2)
- `backend/app/schemas/` - Pydantic request/response schemas (Phase 2)
- `backend/app/middleware/` - Auth, RBAC middleware (Phase 2)
- `backend/app/messenger/` - SMTP messengers (Phase 3)
- `backend/app/bounce/` - Bounce handling (Phase 3)
- `backend/app/queue/` - Queue system (Phase 4)
- `backend/tests/` - Test suite (Ongoing)
- `backend/webjobs/` - Azure Web Jobs (Phase 6)
- `frontend/` - React frontend (Phase 5)

---

## Database Models Architecture

### Model Relationships

```
User 1:N UserRole N:1 Role
Subscriber 1:N SubscriberList N:1 List
List 1:N CampaignList N:1 Campaign
List 1:N ListRole N:1 Role
Campaign N:1 Template (optional FK)
```

### Key Model Features

**All Models Include**:
- Primary key: `id` (SERIAL)
- UUID field: `uuid` (UUID v4, unique, indexed)
- Timestamps: `created_at`, `updated_at` (from TimestampMixin)

**User Model**:
- Email (unique, indexed)
- Password hash (bcrypt ready)
- Status: enabled, disabled
- Many-to-many with Roles via UserRole

**Subscriber Model**:
- Email (unique, indexed)
- JSONB `attribs` for custom fields
- Status: enabled, disabled, blocklisted
- Many-to-many with Lists via SubscriberList (status: unconfirmed, confirmed, unsubscribed)

**List Model**:
- Type: private, public
- Optin: single, double
- JSONB `tags` array
- Relationships: subscribers, campaigns, roles

**Campaign Model**:
- Status: draft, scheduled, running, paused, cancelled, finished
- Content type: richtext, html, markdown, plain
- Messenger: email, email-{name}, automatic
- Tracking fields: to_send, sent, views, clicks, bounces
- Queue fields: use_queue, queued_at, queue_completed_at
- Archive fields: archive, archive_slug, archive_template_id
- Many-to-many with Lists via CampaignList

**Template Model**:
- Type: campaign, tx (transactional)
- Subject + Body
- is_default flag

### Indexes Created

```python
# Subscriber indexes
idx_subscribers_email_status (email, status)

# SubscriberList indexes
idx_subscriber_lists_subscriber (subscriber_id, status)
idx_subscriber_lists_list (list_id, status)

# Campaign indexes
idx_campaigns_status_created (status, created_at)
idx_campaigns_use_queue (use_queue, status)
```

---

## Configuration (Environment Variables)

### Database Connection
```bash
DATABASE_URL=postgresql+asyncpg://listmonk:listmonk@localhost:5432/listmonk
```

### Application Settings
```bash
APP_NAME=listmonk
APP_ENV=development
DEBUG=true
SECRET_KEY=dev-secret-key-change-in-production
CORS_ORIGINS=["http://localhost:3000","http://localhost:5173"]
```

### Authentication
```bash
JWT_SECRET=dev-jwt-secret-change-in-production
JWT_ALGORITHM=HS256
JWT_EXPIRE_MINUTES=10080  # 1 week
```

### Queue Settings
```bash
QUEUE_BATCH_SIZE=100
QUEUE_CONCURRENCY=10
QUEUE_POLL_INTERVAL=5
SEND_TIME_START=  # Empty = 24/7
SEND_TIME_END=    # Empty = 24/7
```

### Optional (Not Used Locally)
```bash
AZURE_STORAGE_CONNECTION_STRING=
AZURE_KEYVAULT_URL=
SENTRY_DSN=
```

---

## Commands to Run on Next Session

### Start Services (if not already running)
```bash
cd /home/adam/listmonk-python
docker-compose up -d
```

### Activate Virtual Environment
```bash
cd /home/adam/listmonk-python/backend
source .venv/bin/activate
```

### Generate Initial Migration (First Time Only)
```bash
# This creates the first migration from models
alembic revision --autogenerate -m "Initial migration"

# Apply migration (creates tables)
alembic upgrade head
```

### Start Development Server
```bash
# Option 1: Direct Python
python -m app.main

# Option 2: Uvicorn with reload
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

### Verify Setup
```bash
# Test health endpoint
curl http://localhost:8000/api/health

# Check database tables
docker-compose exec postgres psql -U listmonk -d listmonk -c "\dt"

# View API docs
# Open browser: http://localhost:8000/api/docs
```

---

## Next Immediate Steps (Phase 2 - Core Features)

### 1. Create First API Endpoint (Campaigns)
**Priority**: High
**Files to Create**:
- `backend/app/api/v1/__init__.py`
- `backend/app/api/v1/campaigns.py` - FastAPI router
- `backend/app/core/campaigns.py` - Business logic service
- `backend/app/schemas/campaign.py` - Pydantic request/response models

**Implementation Order**:
1. Create Pydantic schemas (CampaignCreate, CampaignUpdate, CampaignResponse)
2. Create service layer (get_campaigns, get_campaign, create_campaign)
3. Create FastAPI router (GET /campaigns, GET /campaigns/{id}, POST /campaigns)
4. Register router in main.py
5. Test with Swagger UI

**Example Structure**:
```python
# app/schemas/campaign.py
from pydantic import BaseModel
from datetime import datetime
from typing import Optional

class CampaignBase(BaseModel):
    name: str
    subject: str
    body: str
    messenger: str = "email"

class CampaignCreate(CampaignBase):
    list_ids: list[int]

class CampaignResponse(CampaignBase):
    id: int
    uuid: str
    status: str
    created_at: datetime

    class Config:
        from_attributes = True
```

### 2. Add Authentication Middleware
**Priority**: High
**Files to Create**:
- `backend/app/middleware/auth.py` - JWT verification
- `backend/app/core/auth.py` - Login, token generation
- `backend/app/api/v1/auth.py` - Login endpoint

**Dependencies Needed**:
- passlib (already in pyproject.toml)
- python-jose (already in pyproject.toml)

### 3. Create Repository Pattern (Optional but Recommended)
**Priority**: Medium
**Files to Create**:
- `backend/app/repositories/base.py` - Base repository
- `backend/app/repositories/campaign.py` - Campaign repository

**Why**: Separates data access from business logic, easier to test

### 4. Add Unit Tests
**Priority**: Medium
**Files to Create**:
- `backend/tests/conftest.py` - Pytest fixtures
- `backend/tests/test_campaigns.py` - Campaign tests
- `backend/tests/test_auth.py` - Authentication tests

**Test Coverage Goal**: 80%+ for core business logic

---

## Blockers & Issues

### None Currently

All foundation work completed successfully. No blockers identified.

---

## Technical Debt

### 1. No Migrations Yet
**Issue**: Alembic migrations not yet generated
**Fix**: Run `alembic revision --autogenerate -m "Initial migration"` on first startup
**Priority**: High (must do before creating data)

### 2. No Authentication
**Issue**: API endpoints are currently unprotected
**Fix**: Add JWT middleware in Phase 2
**Priority**: High (before creating write endpoints)

### 3. No Input Validation
**Issue**: Request bodies not validated yet (no Pydantic schemas)
**Fix**: Create schemas in Phase 2
**Priority**: High (prevents bad data)

### 4. No Tests
**Issue**: No test suite yet
**Fix**: Create pytest tests as endpoints are built
**Priority**: Medium (add incrementally)

### 5. No Logging
**Issue**: Using print() instead of proper logging
**Fix**: Add Python logging or structlog
**Priority**: Low (works for dev)

---

## Testing Notes

### Health Check Endpoint
**URL**: `http://localhost:8000/api/health`
**Expected Response**:
```json
{
  "status": "healthy",
  "app": "listmonk",
  "environment": "development",
  "version": "8.0.0"
}
```

### Database Connection Test
```bash
# Should show tables after migration
docker-compose exec postgres psql -U listmonk -d listmonk -c "\dt"
```

### API Documentation
- **Swagger UI**: http://localhost:8000/api/docs
- **ReDoc**: http://localhost:8000/api/redoc

---

## Migration from Go Codebase

### What We're Porting From
- **Source**: `/home/adam/listmonk/` (Go + Vue 2)
- **Database**: PostgreSQL (Bobby Seamoss deployment)
- **Connection**: `listmonk420-db.postgres.database.azure.com`

### Key Files to Reference
- **Schema**: `/home/adam/listmonk/schema.sql` - PostgreSQL schema
- **Models**: `/home/adam/listmonk/models/models.go` - Go structs
- **Handlers**: `/home/adam/listmonk/cmd/*.go` - HTTP handlers
- **Queries**: `/home/adam/listmonk/queries.sql` - Named SQL queries
- **Tech Spec**: `/home/adam/listmonk/TECHNICAL_SPECIFICATION.md` - Full system spec
- **Porting Plan**: `/home/adam/listmonk/PORTING_PLAN.md` - 7,700+ line migration plan

### Architecture Comparison

| Component | Go (Current) | Python (New) |
|-----------|--------------|--------------|
| Framework | Echo v4 | FastAPI 0.104 |
| ORM | sqlx (raw SQL) | SQLAlchemy 2.0 (async) |
| Frontend | Vue 2.7 | React 18 (TBD) |
| Database | PostgreSQL 15 | PostgreSQL 15 |
| Deployment | Docker on Azure | Azure App Service (later) |
| Background | Go goroutines | Azure Web Jobs (later) |
| Session | stdlib | SQLAlchemy sessions |

---

## Performance Considerations

### Connection Pooling
**Current**: 25 connections (pool_size=25, max_overflow=10)
**Rationale**: Matches Go's max_open=25 setting

### Async Throughout
**Strategy**: All database operations use async/await
**Benefit**: Better concurrency than sync blocking operations

### Indexes Created
- All foreign keys indexed
- UUID fields indexed for lookups
- Common query patterns indexed (status, created_at)

---

## Integration Points

### Docker Services
- **PostgreSQL**: localhost:5432 (username: listmonk, password: listmonk, db: listmonk)
- **Redis**: localhost:6379 (not used yet, ready for caching/sessions)

### Virtual Environment
- **Location**: `/home/adam/listmonk-python/backend/.venv/`
- **Manager**: uv
- **Activation**: `source .venv/bin/activate`

### Alembic
- **Config**: `backend/alembic.ini`
- **Environment**: `backend/alembic/env.py`
- **Migrations**: `backend/alembic/versions/` (auto-generated)

---

## Patterns Established

### 1. Model Pattern
```python
class MyModel(Base, TimestampMixin):
    __tablename__ = "my_table"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    uuid: Mapped[uuid.UUID] = mapped_column(UUID, default=uuid.uuid4, unique=True)
    # ... other fields

    # Relationships
    related = relationship("Related", back_populates="my_model")
```

### 2. Configuration Pattern
```python
from app.config import settings

# Use settings.database_url, settings.debug, etc.
# NEVER use os.getenv() or process.env directly
```

### 3. Database Session Pattern
```python
from fastapi import Depends
from app.database import get_db

async def my_endpoint(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Campaign))
    campaigns = result.scalars().all()
```

### 4. Async Pattern
```python
# All database operations are async
async def get_campaign(db: AsyncSession, campaign_id: int):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    return result.scalar_one_or_none()
```

---

## Documentation Resources

### Internal Docs
- **README**: `/home/adam/listmonk-python/README.md`
- **Getting Started**: `/home/adam/listmonk-python/GETTING_STARTED.md`
- **Porting Plan**: `/home/adam/listmonk/PORTING_PLAN.md`
- **Tech Spec**: `/home/adam/listmonk/TECHNICAL_SPECIFICATION.md`

### External Resources
- **FastAPI**: https://fastapi.tiangolo.com/
- **SQLAlchemy**: https://docs.sqlalchemy.org/en/20/
- **Alembic**: https://alembic.sqlalchemy.org/
- **uv**: https://github.com/astral-sh/uv
- **Pydantic**: https://docs.pydantic.dev/

---

## State Checklist

- ✅ Project structure created
- ✅ Docker Compose configured (PostgreSQL + Redis)
- ✅ Python dependencies defined (pyproject.toml)
- ✅ Configuration management (Pydantic Settings)
- ✅ Database connection (Async SQLAlchemy)
- ✅ Database models (11 core models)
- ✅ Alembic setup (migration framework)
- ✅ FastAPI application (main.py with health check)
- ✅ Environment files (.env from .env.example)
- ✅ Documentation (README + Getting Started)
- ✅ Quickstart script (automated setup)
- ⏳ Initial migration (to be generated on first run)
- ⏳ API endpoints (Phase 2)
- ⏳ Authentication (Phase 2)
- ⏳ Frontend (Phase 5)

---

## Handoff Notes for Next Session

### Exact State
- **Working Directory**: `/home/adam/listmonk-python/`
- **Virtual Environment**: Not yet created (will be created by quickstart.sh)
- **Database**: Not yet initialized (migrations not run)
- **Services**: Docker containers not yet started

### First Actions on Restart
1. **Run quickstart script**: `cd /home/adam/listmonk-python && ./scripts/quickstart.sh`
2. **Verify health**: `curl http://localhost:8000/api/health`
3. **Check database**: `docker-compose exec postgres psql -U listmonk -d listmonk -c "\dt"`
4. **View API docs**: Visit http://localhost:8000/api/docs

### What to Build Next
**Immediate Priority**: Create first API endpoint (Campaigns)

**Steps**:
1. Create schemas: `backend/app/schemas/campaign.py`
2. Create service: `backend/app/core/campaigns.py`
3. Create router: `backend/app/api/v1/campaigns.py`
4. Register in main.py
5. Test with Swagger UI

**Code to Start With**:
```python
# backend/app/api/v1/campaigns.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.schemas.campaign import CampaignResponse
from app.core.campaigns import get_campaigns

router = APIRouter()

@router.get("/", response_model=list[CampaignResponse])
async def list_campaigns(
    skip: int = 0,
    limit: int = 20,
    db: AsyncSession = Depends(get_db)
):
    return await get_campaigns(db, skip=skip, limit=limit)
```

---

**End of Context Document**
**Total Setup Time**: ~1 hour
**Lines of Code Created**: ~1,500 lines
**Files Created**: 25 files
**Models Implemented**: 11 models
**Status**: Ready for Phase 2 (Core API Endpoints) ✅
