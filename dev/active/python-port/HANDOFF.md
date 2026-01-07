# Python Port - Session Handoff

**Session**: 1 (Initial Setup)
**Date**: 2025-11-13
**Duration**: ~1 hour
**Status**: Foundation Complete ✅

---

## 🎯 What Was Accomplished

Successfully completed **Phase 1 (Foundation)** of the Python port:

1. ✅ Created complete project structure at `/home/adam/listmonk-python/`
2. ✅ Configured `uv` for Python package management (Kali Linux compatible)
3. ✅ Setup Docker Compose for local PostgreSQL + Redis
4. ✅ Created 11 core database models (SQLAlchemy 2.0 async)
5. ✅ Configured Alembic for database migrations
6. ✅ Created FastAPI application with health check
7. ✅ Setup Pydantic Settings for configuration
8. ✅ Wrote comprehensive documentation (README + Getting Started)
9. ✅ Created automated quickstart script

**No errors encountered** - everything built successfully.

---

## 📁 Project Location

```
/home/adam/listmonk-python/
```

All work is in this new directory. Original Go project remains untouched at `/home/adam/listmonk/`.

---

## 🚀 How to Resume Work

### Option 1: Automated (Recommended)

```bash
cd /home/adam/listmonk-python
./scripts/quickstart.sh
```

This script will:
- ✓ Check prerequisites
- ✓ Start Docker services
- ✓ Create virtual environment
- ✓ Install dependencies
- ✓ Configure environment
- ✓ Run migrations
- ✓ Verify setup

### Option 2: Manual

```bash
cd /home/adam/listmonk-python

# Start services
docker-compose up -d

# Setup Python environment
cd backend
uv venv
source .venv/bin/activate
uv pip install -e ".[dev]"

# Create initial migration
alembic revision --autogenerate -m "Initial migration"
alembic upgrade head

# Start server
python -m app.main
```

### Verify Setup

```bash
# Test health endpoint
curl http://localhost:8000/api/health

# Expected response:
# {"status":"healthy","app":"listmonk","environment":"development","version":"8.0.0"}

# Check database tables
docker-compose exec postgres psql -U listmonk -d listmonk -c "\dt"

# View API docs
# Open browser: http://localhost:8000/api/docs
```

---

## 📝 What to Build Next

### Immediate Priority: Campaign API (Phase 2)

**Goal**: Create first working API endpoint

**Steps** (in order):

#### 1. Create Pydantic Schemas

**File**: `backend/app/schemas/campaign.py`

```python
from pydantic import BaseModel
from datetime import datetime
from typing import Optional
from uuid import UUID

class CampaignBase(BaseModel):
    name: str
    subject: str
    body: str
    from_email: Optional[str] = None
    messenger: str = "email"
    content_type: str = "richtext"
    tags: list[str] = []

class CampaignCreate(CampaignBase):
    list_ids: list[int]
    template_id: Optional[int] = None

class CampaignUpdate(BaseModel):
    name: Optional[str] = None
    subject: Optional[str] = None
    body: Optional[str] = None
    status: Optional[str] = None

class CampaignResponse(CampaignBase):
    id: int
    uuid: UUID
    status: str
    to_send: int
    sent: int
    views: int
    clicks: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True

class CampaignListResponse(BaseModel):
    items: list[CampaignResponse]
    total: int
    page: int
    per_page: int
```

#### 2. Create Service Layer

**File**: `backend/app/core/campaigns.py`

```python
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from app.models.campaign import Campaign
from app.schemas.campaign import CampaignCreate, CampaignUpdate
from typing import Optional

async def get_campaigns(
    db: AsyncSession,
    skip: int = 0,
    limit: int = 20,
    status: Optional[str] = None
) -> list[Campaign]:
    """Get list of campaigns with optional filtering."""
    query = select(Campaign)

    if status:
        query = query.where(Campaign.status == status)

    query = query.offset(skip).limit(limit).order_by(Campaign.created_at.desc())

    result = await db.execute(query)
    return result.scalars().all()

async def get_campaign(db: AsyncSession, campaign_id: int) -> Optional[Campaign]:
    """Get single campaign by ID."""
    result = await db.execute(
        select(Campaign).where(Campaign.id == campaign_id)
    )
    return result.scalar_one_or_none()

async def create_campaign(
    db: AsyncSession,
    campaign: CampaignCreate
) -> Campaign:
    """Create new campaign."""
    db_campaign = Campaign(
        name=campaign.name,
        subject=campaign.subject,
        body=campaign.body,
        from_email=campaign.from_email,
        messenger=campaign.messenger,
        content_type=campaign.content_type,
        tags=campaign.tags,
        status="draft"
    )

    db.add(db_campaign)
    await db.commit()
    await db.refresh(db_campaign)

    return db_campaign

async def update_campaign(
    db: AsyncSession,
    campaign_id: int,
    updates: CampaignUpdate
) -> Optional[Campaign]:
    """Update campaign."""
    campaign = await get_campaign(db, campaign_id)
    if not campaign:
        return None

    update_data = updates.model_dump(exclude_unset=True)
    for key, value in update_data.items():
        setattr(campaign, key, value)

    await db.commit()
    await db.refresh(campaign)

    return campaign
```

#### 3. Create API Router

**File**: `backend/app/api/v1/campaigns.py`

```python
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.schemas.campaign import (
    CampaignCreate,
    CampaignUpdate,
    CampaignResponse,
    CampaignListResponse
)
from app.core import campaigns as campaign_service

router = APIRouter()

@router.get("/", response_model=CampaignListResponse)
async def list_campaigns(
    skip: int = 0,
    limit: int = 20,
    status: str = None,
    db: AsyncSession = Depends(get_db)
):
    """List all campaigns."""
    campaigns = await campaign_service.get_campaigns(db, skip, limit, status)
    total = len(campaigns)  # TODO: Add count query

    return {
        "items": campaigns,
        "total": total,
        "page": skip // limit + 1,
        "per_page": limit
    }

@router.post("/", response_model=CampaignResponse, status_code=201)
async def create_campaign(
    campaign: CampaignCreate,
    db: AsyncSession = Depends(get_db)
):
    """Create a new campaign."""
    return await campaign_service.create_campaign(db, campaign)

@router.get("/{campaign_id}", response_model=CampaignResponse)
async def get_campaign(
    campaign_id: int,
    db: AsyncSession = Depends(get_db)
):
    """Get a single campaign by ID."""
    campaign = await campaign_service.get_campaign(db, campaign_id)
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return campaign

@router.put("/{campaign_id}", response_model=CampaignResponse)
async def update_campaign(
    campaign_id: int,
    updates: CampaignUpdate,
    db: AsyncSession = Depends(get_db)
):
    """Update a campaign."""
    campaign = await campaign_service.update_campaign(db, campaign_id, updates)
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return campaign
```

#### 4. Register Router in Main App

**File**: `backend/app/main.py`

Add after the existing imports:

```python
from app.api.v1 import campaigns

# ... existing code ...

# Register routers
app.include_router(
    campaigns.router,
    prefix="/api/v1/campaigns",
    tags=["campaigns"]
)
```

#### 5. Test Endpoints

```bash
# Start server
cd /home/adam/listmonk-python/backend
source .venv/bin/activate
python -m app.main

# Open Swagger UI
# Browser: http://localhost:8000/api/docs

# Test creating a campaign
curl -X POST http://localhost:8000/api/v1/campaigns \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Campaign",
    "subject": "Hello World",
    "body": "<p>Test email</p>",
    "list_ids": []
  }'

# Test listing campaigns
curl http://localhost:8000/api/v1/campaigns
```

---

## 🔑 Key Files Reference

### Configuration
- **Environment**: `backend/.env`
- **Dependencies**: `backend/pyproject.toml`
- **Alembic**: `backend/alembic.ini`

### Application Core
- **Main App**: `backend/app/main.py`
- **Database**: `backend/app/database.py`
- **Config**: `backend/app/config.py`

### Models (All Created)
- `backend/app/models/base.py`
- `backend/app/models/user.py`
- `backend/app/models/subscriber.py`
- `backend/app/models/list.py`
- `backend/app/models/campaign.py`
- `backend/app/models/template.py`

### To Be Created (Next Session)
- `backend/app/schemas/campaign.py` - Pydantic schemas
- `backend/app/core/campaigns.py` - Business logic
- `backend/app/api/v1/__init__.py` - API package
- `backend/app/api/v1/campaigns.py` - API router

---

## 🗄️ Database Info

### Connection Details
```bash
Host: localhost
Port: 5432
Database: listmonk
Username: listmonk
Password: listmonk
```

### Connection String
```
postgresql+asyncpg://listmonk:listmonk@localhost:5432/listmonk
```

### Access Database
```bash
# Via Docker
docker-compose exec postgres psql -U listmonk -d listmonk

# Via local psql
PGPASSWORD=listmonk psql -h localhost -U listmonk -d listmonk
```

### Common SQL Commands
```sql
-- List tables
\dt

-- Describe campaign table
\d campaigns

-- View all campaigns
SELECT id, name, status, created_at FROM campaigns;

-- Count campaigns
SELECT COUNT(*) FROM campaigns;

-- Exit
\q
```

---

## 🧪 Testing Commands

```bash
# Activate venv first
cd /home/adam/listmonk-python/backend
source .venv/bin/activate

# Run all tests
pytest

# Run with coverage
pytest --cov=app --cov-report=html

# Run specific test file
pytest tests/test_campaigns.py

# Run with verbose output
pytest -v

# View coverage report
firefox htmlcov/index.html
```

---

## 📚 Documentation Files

### In Project
- `/home/adam/listmonk-python/README.md` - Project overview
- `/home/adam/listmonk-python/GETTING_STARTED.md` - Setup guide
- `/home/adam/listmonk-python/backend/.env.example` - Config template

### In Original Project (Reference)
- `/home/adam/listmonk/PORTING_PLAN.md` - Complete 7,700+ line porting plan
- `/home/adam/listmonk/TECHNICAL_SPECIFICATION.md` - Current system spec
- `/home/adam/listmonk/schema.sql` - PostgreSQL schema
- `/home/adam/listmonk/queries.sql` - Named SQL queries

### In Dev Docs (This Session)
- `/home/adam/listmonk/dev/active/python-port/python-port-context.md` - Full context
- `/home/adam/listmonk/dev/active/python-port/python-port-tasks.md` - Task tracker
- `/home/adam/listmonk/dev/active/python-port/HANDOFF.md` - This file

---

## ⚠️ Important Notes

### Virtual Environment
**ALWAYS activate venv before working**:
```bash
cd /home/adam/listmonk-python/backend
source .venv/bin/activate
```

You'll know it's activated when you see `(.venv)` in your prompt.

### Docker Services
**ALWAYS ensure Docker services are running**:
```bash
docker-compose ps

# If not running:
docker-compose up -d
```

### Migrations
**First time only**: Generate and run initial migration:
```bash
alembic revision --autogenerate -m "Initial migration"
alembic upgrade head
```

After that, migrations run automatically on app startup.

### No Authentication Yet
Current API endpoints are **unprotected**. Authentication will be added in Phase 6 (Weeks 35-36).

---

## 🐛 Known Issues

**None**. Everything built successfully without errors.

---

## 📊 Progress Tracking

```
Foundation:  ████████████████████ 100%
Phase 2:     ░░░░░░░░░░░░░░░░░░░░   0%
Phase 3:     ░░░░░░░░░░░░░░░░░░░░   0%
Overall:     ██░░░░░░░░░░░░░░░░░░  11%
```

**Phases**: 9 total
**Completed**: Phase 1 (Foundation)
**Current**: Phase 2 (Core Features - Campaign API)
**Next**: Create campaign schemas, service, and router

---

## 💡 Tips for Next Session

1. **Start with quickstart.sh** - It does everything automatically
2. **Use Swagger UI** - http://localhost:8000/api/docs for API testing
3. **Check logs** - Watch uvicorn output for errors
4. **Use docker-compose logs** - If database issues arise
5. **Read python-port-context.md** - Has all technical details
6. **Follow the porting plan** - Reference PORTING_PLAN.md for Go code examples

---

## 🎯 Success Criteria for Next Session

By end of next session, should have:
- [x] Campaign API endpoints working (GET, POST, PUT)
- [x] Able to create campaigns via API
- [x] Able to list campaigns via API
- [x] Swagger UI shows campaign endpoints
- [x] At least 1 unit test for campaign service

---

## 🚀 Commands Quick Reference

```bash
# Start everything
cd /home/adam/listmonk-python
./scripts/quickstart.sh

# Start just services
docker-compose up -d

# Activate venv
cd backend && source .venv/bin/activate

# Start API server
python -m app.main

# Run tests
pytest

# Create migration
alembic revision --autogenerate -m "message"

# Apply migrations
alembic upgrade head

# Database access
docker-compose exec postgres psql -U listmonk -d listmonk

# View logs
docker-compose logs -f postgres
docker-compose logs -f redis

# Stop everything
docker-compose down
```

---

**End of Handoff**

**Status**: Ready for Phase 2 ✅
**Blockers**: None
**Next Steps**: Create campaign API (schemas → service → router → test)
**Timeline**: On track for 56-week completion

Good luck! 🎉
