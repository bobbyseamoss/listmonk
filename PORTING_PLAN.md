# Comprehensive Porting Plan: listmonk → Python/React/Azure

**Document Version**: 1.0
**Date**: 2025-11-13
**Scope**: Full feature parity port from Go/Vue2 to Python/React

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Design](#2-architecture-design)
3. [Database Migration Plan](#3-database-migration-plan)
4. [Backend Porting Plan](#4-backend-porting-plan)
5. [Frontend Porting Plan](#5-frontend-porting-plan)
6. [Advanced Features Porting](#6-advanced-features-porting)
7. [Azure Deployment Architecture](#7-azure-deployment-architecture)
8. [Development Phases](#8-development-phases)
9. [Technology Stack Details](#9-technology-stack-details)
10. [API Endpoint Mapping](#10-api-endpoint-mapping)
11. [Component Mapping](#11-component-mapping)
12. [Background Jobs Architecture](#12-background-jobs-architecture)
13. [Testing Strategy](#13-testing-strategy)
14. [Migration Strategy](#14-migration-strategy)
15. [Risks and Mitigation](#15-risks-and-mitigation)

---

## 1. EXECUTIVE SUMMARY

### 1.1 Project Overview

**Objective**: Port the entire listmonk newsletter/mailing list management system from Go/Vue2 to Python/React while maintaining full feature parity.

**Current Architecture** (listmonk v7.x):
- Backend: Go 1.x with Echo v4 framework (~10,130 lines)
- Frontend: Vue 2.7 with Buefy (~60 files, 15k-20k lines)
- Database: PostgreSQL (30+ tables, sophisticated schema)
- Distribution: Single binary with embedded assets
- Deployment: Docker containers

**Target Architecture**:
- Backend: Python 3.11+ with FastAPI
- Frontend: React 18+ with TypeScript
- Database: PostgreSQL (same schema, Python ORM)
- Hosting: Azure App Service (Linux, no containers)
- Background Jobs: Azure Web Jobs
- Storage: Azure Blob Storage

### 1.2 Technology Stack Comparison

| Component | Current (Go/Vue2) | Target (Python/React) |
|-----------|-------------------|----------------------|
| **Backend Language** | Go 1.x | Python 3.11+ |
| **Web Framework** | Echo v4 | FastAPI 0.104+ |
| **ORM** | sqlx (raw SQL) | SQLAlchemy 2.0+ (async) |
| **Migrations** | Go code in internal/migrations/ | Alembic |
| **Template Engine** | Go html/template | Jinja2 |
| **Email Sending** | net/smtp | aiosmtplib (async) |
| **Frontend Framework** | Vue 2.7 | React 18+ with TypeScript |
| **State Management** | Vuex 3 | Zustand or Redux Toolkit |
| **UI Library** | Buefy (Bulma) | Material-UI v5 or Ant Design |
| **Build Tool** | Vite | Vite |
| **Routing** | Vue Router 3 | React Router v6 |
| **HTTP Client** | Axios | Axios + TanStack Query |
| **Asset Embedding** | stuffbin | Azure Blob Storage |
| **Deployment** | Docker | Azure App Service |
| **Background Jobs** | Goroutines | Azure Web Jobs |
| **Configuration** | koanf (TOML/ENV) | pydantic-settings (ENV) |

### 1.3 Azure Services Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Azure Subscription                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure App Service (Linux, Python 3.11)                     │ │
│  │ ┌────────────────────────────────────────────────────────┐ │ │
│  │ │ FastAPI Application                                    │ │ │
│  │ │ - REST API (156+ endpoints)                           │ │ │
│  │ │ - Authentication & RBAC                               │ │ │
│  │ │ - Business Logic                                      │ │ │
│  │ │ - Static Files (React build)                         │ │ │
│  │ └────────────────────────────────────────────────────────┘ │ │
│  │                                                              │ │
│  │ Azure Web Jobs (within App Service)                        │ │
│  │ ┌──────────────┐ ┌──────────────┐ ┌─────────────────────┐ │ │
│  │ │ Campaign     │ │ Queue        │ │ Stats Sync          │ │ │
│  │ │ Processor    │ │ Processor    │ │ (every 5 min)       │ │ │
│  │ │ (continuous) │ │ (continuous) │ │                     │ │ │
│  │ └──────────────┘ └──────────────┘ └─────────────────────┘ │ │
│  │ ┌──────────────┐ ┌──────────────────────────────────────┐ │ │
│  │ │ Auto-Pause   │ │ Bounce Mailbox Scanner               │ │ │
│  │ │ Scheduler    │ │ (every N minutes)                    │ │ │
│  │ │ (every 1 min)│ │                                       │ │ │
│  │ └──────────────┘ └──────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure Database for PostgreSQL (Flexible Server)           │ │
│  │ - 30+ tables (subscribers, campaigns, lists, etc.)        │ │
│  │ - Materialized views for performance                      │ │
│  │ - Full-text search                                        │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure Blob Storage                                         │ │
│  │ - Media uploads (images, attachments)                     │ │
│  │ - Campaign archives                                       │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure Key Vault                                            │ │
│  │ - SMTP credentials                                        │ │
│  │ - Database passwords                                      │ │
│  │ - API keys (SendGrid, Shopify, etc.)                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure Application Insights                                 │ │
│  │ - Request/response logging                                │ │
│  │ - Error tracking                                          │ │
│  │ - Performance metrics                                     │ │
│  │ - Custom events                                           │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Azure Event Grid (optional - for webhooks)                │ │
│  │ - Email delivery events                                   │ │
│  │ - Email engagement events                                 │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 Key Architecture Decisions

**1. FastAPI vs Django**
- **Choice**: FastAPI
- **Rationale**:
  - Native async support (critical for SMTP, email sending)
  - Better performance (on par with Go)
  - Modern Python features (type hints, Pydantic)
  - Automatic OpenAPI documentation
  - Easier WebSocket support (for real-time updates)
  - Lightweight and flexible

**2. SQLAlchemy 2.0 Async**
- Native async/await support
- Type-safe queries
- Alembic for migrations
- Compatible with PostgreSQL advanced features

**3. React vs Vue 3**
- **Choice**: React 18+
- **Rationale**:
  - Larger ecosystem and community
  - Better TypeScript support
  - More mature component libraries
  - Easier to find developers
  - Server Components (future-proofing)

**4. State Management**
- **Choice**: Zustand (recommended) or Redux Toolkit
- **Rationale**:
  - Zustand: Simpler, less boilerplate, TypeScript-first
  - Redux Toolkit: More features, better DevTools, familiar

**5. Azure Web Jobs vs Azure Functions**
- **Choice**: Azure Web Jobs
- **Rationale**:
  - Part of App Service (no additional service)
  - Continuous jobs for campaign/queue processing
  - Scheduled jobs for maintenance
  - Shared environment with web app
  - Lower cost

**6. No Containerization**
- Direct deployment to Azure App Service
- Python runtime managed by Azure
- Simpler deployment process
- Auto-scaling built-in

### 1.5 Timeline and Resource Estimates

**Total Estimated Timeline**: 38-56 weeks (9-14 months)

**Team Composition** (recommended):
- 1 Senior Backend Developer (Python/FastAPI)
- 1 Senior Frontend Developer (React/TypeScript)
- 1 Full-Stack Developer (Python + React)
- 1 DevOps Engineer (Azure)
- 1 QA Engineer
- 1 Technical Lead / Architect (part-time)

**Phase Breakdown**:
1. Foundation: 4-6 weeks
2. Core Features: 6-8 weeks
3. Advanced Campaign Features: 4-6 weeks
4. Queue System: 6-8 weeks
5. Advanced Analytics: 4-6 weeks
6. Admin Features: 3-4 weeks
7. Import/Export & Public Pages: 3-4 weeks
8. Testing & Optimization: 4-6 weeks
9. Deployment & Migration: 2-4 weeks

**Budget Considerations**:
- Azure App Service: ~$100-500/month (depends on tier)
- Azure Database for PostgreSQL: ~$100-300/month
- Azure Blob Storage: ~$20-50/month
- Application Insights: ~$50-100/month
- Total Azure: ~$270-950/month

### 1.6 Success Criteria

**Functional Requirements**:
- ✅ All 156+ API endpoints implemented
- ✅ All 30+ database tables migrated
- ✅ All frontend pages and components ported
- ✅ Queue-based email delivery system operational
- ✅ Multi-SMTP support (30+ servers)
- ✅ Azure Event Grid integration
- ✅ Shopify integration
- ✅ All analytics features working
- ✅ RBAC system implemented
- ✅ Import/export functionality

**Performance Requirements**:
- API response time: <200ms (p95)
- Campaign send rate: 500-1000 emails/second (with queue)
- Queue processing: Poll every 5 seconds
- Database query time: <100ms (p95)
- Frontend load time: <2 seconds
- Support 1M+ subscribers

**Quality Requirements**:
- Unit test coverage: >80%
- Integration test coverage: >60%
- E2E test coverage: Critical user flows
- Zero data loss during migration
- 99.9% uptime SLA

---

## 2. ARCHITECTURE DESIGN

### 2.1 Backend Architecture (FastAPI)

**Project Structure**:
```
listmonk-python/
├── app/
│   ├── __init__.py
│   ├── main.py                    # FastAPI application entry
│   ├── config.py                  # Configuration management
│   ├── dependencies.py            # Dependency injection
│   │
│   ├── api/                       # API routes
│   │   ├── __init__.py
│   │   ├── v1/
│   │   │   ├── __init__.py
│   │   │   ├── campaigns.py
│   │   │   ├── subscribers.py
│   │   │   ├── lists.py
│   │   │   ├── templates.py
│   │   │   ├── media.py
│   │   │   ├── queue.py
│   │   │   ├── bounces.py
│   │   │   ├── users.py
│   │   │   ├── settings.py
│   │   │   ├── dashboard.py
│   │   │   └── public.py
│   │   └── dependencies.py        # Route dependencies
│   │
│   ├── core/                      # Business logic
│   │   ├── __init__.py
│   │   ├── campaigns.py
│   │   ├── subscribers.py
│   │   ├── lists.py
│   │   ├── templates.py
│   │   ├── bounces.py
│   │   ├── media.py
│   │   ├── users.py
│   │   ├── roles.py
│   │   └── settings.py
│   │
│   ├── models/                    # SQLAlchemy models
│   │   ├── __init__.py
│   │   ├── base.py
│   │   ├── subscriber.py
│   │   ├── list.py
│   │   ├── campaign.py
│   │   ├── template.py
│   │   ├── media.py
│   │   ├── bounce.py
│   │   ├── user.py
│   │   ├── role.py
│   │   └── queue.py
│   │
│   ├── schemas/                   # Pydantic schemas
│   │   ├── __init__.py
│   │   ├── subscriber.py
│   │   ├── list.py
│   │   ├── campaign.py
│   │   ├── template.py
│   │   └── ...
│   │
│   ├── db/                        # Database
│   │   ├── __init__.py
│   │   ├── session.py            # Database session
│   │   └── base.py               # Base model
│   │
│   ├── manager/                   # Campaign execution
│   │   ├── __init__.py
│   │   ├── campaign_manager.py
│   │   ├── message.py
│   │   └── pipeline.py
│   │
│   ├── messenger/                 # Message delivery
│   │   ├── __init__.py
│   │   ├── base.py
│   │   ├── email.py              # SMTP messenger
│   │   ├── postback.py           # HTTP webhook
│   │   └── automatic.py          # Queue-based
│   │
│   ├── queue/                     # Queue processor
│   │   ├── __init__.py
│   │   ├── processor.py
│   │   ├── calculator.py
│   │   └── scheduler.py
│   │
│   ├── bounce/                    # Bounce handling
│   │   ├── __init__.py
│   │   ├── manager.py
│   │   ├── mailbox.py            # POP3 scanner
│   │   └── webhooks/
│   │       ├── __init__.py
│   │       ├── ses.py
│   │       ├── sendgrid.py
│   │       ├── postmark.py
│   │       ├── azure.py
│   │       └── shopify.py
│   │
│   ├── auth/                      # Authentication
│   │   ├── __init__.py
│   │   ├── jwt.py
│   │   ├── oidc.py
│   │   └── permissions.py
│   │
│   ├── services/                  # External services
│   │   ├── __init__.py
│   │   ├── blob_storage.py       # Azure Blob
│   │   ├── email.py              # Email sending
│   │   ├── analytics.py          # Tracking
│   │   └── azure_events.py       # Event Grid
│   │
│   ├── utils/                     # Utilities
│   │   ├── __init__.py
│   │   ├── crypto.py
│   │   ├── validators.py
│   │   └── helpers.py
│   │
│   └── templates/                 # Jinja2 templates
│       ├── email/
│       └── public/
│
├── webjobs/                       # Azure Web Jobs
│   ├── campaign_processor/
│   │   ├── __init__.py
│   │   └── run.py
│   ├── queue_processor/
│   │   ├── __init__.py
│   │   └── run.py
│   ├── stats_sync/
│   │   ├── __init__.py
│   │   └── run.py
│   └── auto_pause/
│       ├── __init__.py
│       └── run.py
│
├── alembic/                       # Database migrations
│   ├── versions/
│   └── env.py
│
├── tests/                         # Tests
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── alembic.ini
├── requirements.txt
├── requirements-dev.txt
└── README.md
```

**FastAPI Application Structure**:

```python
# app/main.py
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles

from app.api.v1 import campaigns, subscribers, lists, templates
from app.config import settings
from app.db.session import engine
from app.models import Base

app = FastAPI(
    title="listmonk",
    description="Newsletter and mailing list manager",
    version="8.0.0"
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(campaigns.router, prefix="/api/v1/campaigns", tags=["campaigns"])
app.include_router(subscribers.router, prefix="/api/v1/subscribers", tags=["subscribers"])
# ... more routers

# Serve React static files
app.mount("/admin", StaticFiles(directory="frontend/build", html=True), name="admin")

@app.on_event("startup")
async def startup():
    # Create tables if not exist (dev only)
    # async with engine.begin() as conn:
    #     await conn.run_sync(Base.metadata.create_all)
    pass

@app.on_event("shutdown")
async def shutdown():
    await engine.dispose()
```

**Async Pattern**:
```python
# app/core/campaigns.py
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from app.models.campaign import Campaign

async def get_campaign(db: AsyncSession, campaign_id: int) -> Campaign:
    result = await db.execute(
        select(Campaign).where(Campaign.id == campaign_id)
    )
    return result.scalar_one_or_none()

async def create_campaign(db: AsyncSession, campaign: CampaignCreate) -> Campaign:
    db_campaign = Campaign(**campaign.dict())
    db.add(db_campaign)
    await db.commit()
    await db.refresh(db_campaign)
    return db_campaign
```

**Dependency Injection**:
```python
# app/dependencies.py
from typing import AsyncGenerator
from fastapi import Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.session import async_session
from app.auth.jwt import get_current_user
from app.models.user import User

async def get_db() -> AsyncGenerator[AsyncSession, None]:
    async with async_session() as session:
        yield session

async def get_current_active_user(
    current_user: User = Depends(get_current_user)
) -> User:
    if not current_user.status == "enabled":
        raise HTTPException(status_code=400, detail="Inactive user")
    return current_user
```

### 2.2 Frontend Architecture (React)

**Project Structure**:
```
frontend/
├── public/
│   ├── index.html
│   └── assets/
├── src/
│   ├── main.tsx                   # App entry point
│   ├── App.tsx                    # Root component
│   ├── vite-env.d.ts
│   │
│   ├── api/                       # API client
│   │   ├── client.ts              # Axios instance
│   │   ├── campaigns.ts
│   │   ├── subscribers.ts
│   │   ├── lists.ts
│   │   └── ...
│   │
│   ├── stores/                    # State management
│   │   ├── authStore.ts
│   │   ├── campaignStore.ts
│   │   ├── subscriberStore.ts
│   │   └── ...
│   │
│   ├── hooks/                     # Custom hooks
│   │   ├── useAuth.ts
│   │   ├── useCampaigns.ts
│   │   ├── useSubscribers.ts
│   │   └── ...
│   │
│   ├── pages/                     # Page components
│   │   ├── Dashboard.tsx
│   │   ├── Campaigns/
│   │   │   ├── CampaignList.tsx
│   │   │   ├── CampaignEditor.tsx
│   │   │   ├── CampaignAnalytics.tsx
│   │   │   └── Queue.tsx
│   │   ├── Subscribers/
│   │   │   ├── SubscriberList.tsx
│   │   │   ├── SubscriberForm.tsx
│   │   │   ├── Import.tsx
│   │   │   └── Bounces.tsx
│   │   ├── Lists/
│   │   ├── Templates/
│   │   ├── Media/
│   │   ├── Settings/
│   │   ├── Users/
│   │   └── Login.tsx
│   │
│   ├── components/                # Reusable components
│   │   ├── Layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── Footer.tsx
│   │   ├── Editors/
│   │   │   ├── RichTextEditor.tsx
│   │   │   ├── CodeEditor.tsx
│   │   │   └── VisualEditor.tsx
│   │   ├── Forms/
│   │   │   ├── ListSelector.tsx
│   │   │   ├── DatePicker.tsx
│   │   │   └── ...
│   │   ├── Charts/
│   │   │   ├── LineChart.tsx
│   │   │   └── BarChart.tsx
│   │   └── Common/
│   │       ├── Button.tsx
│   │       ├── Table.tsx
│   │       ├── Modal.tsx
│   │       └── ...
│   │
│   ├── types/                     # TypeScript types
│   │   ├── campaign.ts
│   │   ├── subscriber.ts
│   │   ├── list.ts
│   │   └── ...
│   │
│   ├── utils/                     # Utilities
│   │   ├── formatters.ts
│   │   ├── validators.ts
│   │   └── constants.ts
│   │
│   └── styles/                    # Global styles
│       └── index.css
│
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

**React Component Pattern**:
```typescript
// src/pages/Campaigns/CampaignList.tsx
import { useEffect } from 'react';
import { useCampaignStore } from '@/stores/campaignStore';
import { DataGrid } from '@mui/material';

export const CampaignList: React.FC = () => {
  const { campaigns, loading, fetchCampaigns } = useCampaignStore();

  useEffect(() => {
    fetchCampaigns();
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h1>Campaigns</h1>
      <DataGrid
        rows={campaigns}
        columns={[...]}
        // ...
      />
    </div>
  );
};
```

**Zustand Store Pattern**:
```typescript
// src/stores/campaignStore.ts
import create from 'zustand';
import { Campaign } from '@/types/campaign';
import * as campaignApi from '@/api/campaigns';

interface CampaignState {
  campaigns: Campaign[];
  loading: boolean;
  error: string | null;
  fetchCampaigns: () => Promise<void>;
  createCampaign: (data: CampaignCreate) => Promise<void>;
  updateCampaign: (id: number, data: CampaignUpdate) => Promise<void>;
  deleteCampaign: (id: number) => Promise<void>;
}

export const useCampaignStore = create<CampaignState>((set) => ({
  campaigns: [],
  loading: false,
  error: null,

  fetchCampaigns: async () => {
    set({ loading: true });
    try {
      const campaigns = await campaignApi.getCampaigns();
      set({ campaigns, loading: false });
    } catch (error) {
      set({ error: error.message, loading: false });
    }
  },

  // ... other actions
}));
```

### 2.3 Database Layer (SQLAlchemy)

**Async Engine Setup**:
```python
# app/db/session.py
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker

from app.config import settings

engine = create_async_engine(
    settings.DATABASE_URL,
    echo=settings.DEBUG,
    pool_size=20,
    max_overflow=10,
)

async_session = sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False
)
```

**Model Definition**:
```python
# app/models/campaign.py
from sqlalchemy import Column, Integer, String, Text, DateTime, Boolean, ARRAY, Enum
from sqlalchemy.dialects.postgresql import UUID, JSONB
from sqlalchemy.orm import relationship
import uuid

from app.models.base import Base

class Campaign(Base):
    __tablename__ = "campaigns"

    id = Column(Integer, primary_key=True)
    uuid = Column(UUID(as_uuid=True), default=uuid.uuid4, unique=True, nullable=False)
    name = Column(String, nullable=False)
    subject = Column(Text, nullable=False)
    from_email = Column(Text, nullable=False)
    body = Column(Text, nullable=False)
    body_source = Column(Text)
    altbody = Column(Text)
    content_type = Column(Enum('richtext', 'html', 'plain', 'markdown', 'visual', name='content_type'))
    status = Column(Enum('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished', name='campaign_status'))
    messenger = Column(Text, nullable=False)
    tags = Column(ARRAY(String(100)))
    headers = Column(JSONB, default=[])
    created_at = Column(DateTime, nullable=False)
    updated_at = Column(DateTime, nullable=False)

    # Queue fields
    use_queue = Column(Boolean, default=False)
    queued_at = Column(DateTime)

    # Relationships
    lists = relationship("CampaignList", back_populates="campaign")
    views = relationship("CampaignView", back_populates="campaign")
```

### 2.4 Background Jobs (Azure Web Jobs)

**Campaign Processor** (continuous):
```python
# webjobs/campaign_processor/run.py
import asyncio
from app.manager.campaign_manager import CampaignManager
from app.db.session import async_session

async def main():
    async with async_session() as db:
        manager = CampaignManager(db)

        while True:
            try:
                # Check for campaigns to process
                await manager.scan_campaigns()

                # Process next batch
                await manager.process_batch()

                # Sleep 1 second
                await asyncio.sleep(1)
            except Exception as e:
                print(f"Error: {e}")
                await asyncio.sleep(5)

if __name__ == "__main__":
    asyncio.run(main())
```

**Queue Processor** (continuous):
```python
# webjobs/queue_processor/run.py
import asyncio
from app.queue.processor import QueueProcessor
from app.db.session import async_session

async def main():
    async with async_session() as db:
        processor = QueueProcessor(db)

        while True:
            try:
                # Poll for queued emails
                await processor.process_batch()

                # Sleep 5 seconds
                await asyncio.sleep(5)
            except Exception as e:
                print(f"Error: {e}")
                await asyncio.sleep(10)

if __name__ == "__main__":
    asyncio.run(main())
```

### 2.5 Authentication & Authorization

**JWT Authentication**:
```python
# app/auth/jwt.py
from datetime import datetime, timedelta
from jose import JWTError, jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials

from app.config import settings
from app.models.user import User
from app.db.session import AsyncSession
from app.dependencies import get_db

security = HTTPBearer()

def create_access_token(data: dict):
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, settings.SECRET_KEY, algorithm="HS256")

async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security),
    db: AsyncSession = Depends(get_db)
) -> User:
    token = credentials.credentials

    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=["HS256"])
        user_id: int = payload.get("sub")
        if user_id is None:
            raise HTTPException(status_code=401, detail="Invalid token")
    except JWTError:
        raise HTTPException(status_code=401, detail="Invalid token")

    # Fetch user from database
    user = await get_user_by_id(db, user_id)
    if user is None:
        raise HTTPException(status_code=401, detail="User not found")

    return user
```

**Permission Decorator**:
```python
# app/auth/permissions.py
from functools import wraps
from fastapi import HTTPException

def require_permission(permission: str):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, current_user, **kwargs):
            if not current_user.has_permission(permission):
                raise HTTPException(status_code=403, detail="Insufficient permissions")
            return await func(*args, current_user=current_user, **kwargs)
        return wrapper
    return decorator

# Usage in route:
@router.post("/")
@require_permission("campaigns:manage")
async def create_campaign(
    campaign: CampaignCreate,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db)
):
    return await create_campaign_logic(db, campaign)
```

---

## 3. DATABASE MIGRATION PLAN

### 3.1 Schema Compatibility

The PostgreSQL schema remains largely the same. Key changes:
- Use SQLAlchemy ORM instead of raw SQL
- Alembic for migrations instead of Go migration files
- Same table structure, indexes, constraints

### 3.2 SQLAlchemy Model Mapping

**All 30+ tables need to be converted to SQLAlchemy models:**

**Example: Subscriber Model**
```python
# app/models/subscriber.py
from sqlalchemy import Column, Integer, String, Text, DateTime, Enum
from sqlalchemy.dialects.postgresql import UUID, JSONB
from sqlalchemy.orm import relationship
import uuid

from app.models.base import Base

class Subscriber(Base):
    __tablename__ = "subscribers"

    id = Column(Integer, primary_key=True, autoincrement=True)
    uuid = Column(UUID(as_uuid=True), default=uuid.uuid4, unique=True, nullable=False)
    email = Column(Text, unique=True, nullable=False)
    name = Column(Text, nullable=False)
    attribs = Column(JSONB, default={})
    status = Column(
        Enum('enabled', 'disabled', 'blocklisted', name='subscriber_status'),
        default='enabled'
    )
    created_at = Column(DateTime, nullable=False)
    updated_at = Column(DateTime, nullable=False)

    # Relationships
    subscriptions = relationship("SubscriberList", back_populates="subscriber")
    bounces = relationship("Bounce", back_populates="subscriber")
    views = relationship("CampaignView", back_populates="subscriber")
    clicks = relationship("LinkClick", back_populates="subscriber")

    # Indexes defined at the end
    __table_args__ = (
        Index('idx_subs_email', func.lower(email), unique=True),
        Index('idx_subs_status', status),
        Index('idx_subs_id_status', id, status),
        Index('idx_subs_created_at', created_at),
        Index('idx_subs_updated_at', updated_at),
    )
```

**Complete Model List** (30+ models needed):
1. Subscriber
2. List
3. SubscriberList (junction table)
4. Campaign
5. CampaignList
6. Template
7. CampaignView
8. Link
9. LinkClick
10. Bounce
11. Media
12. CampaignMedia
13. User
14. Role
15. Session
16. EmailQueue
17. SMTPDailyUsage
18. SMTPRateLimitState
19. AccountRateLimitState
20. SubscriberLastSend
21. AzureDeliveryEvent
22. AzureEngagementEvent
23. PurchaseAttribution
24. WebhookLog
25. Settings
26. MatDashboardCounts (materialized view)
27. MatDashboardCharts (materialized view)
28. MatListSubscriberStats (materialized view)

### 3.3 Alembic Migration Setup

**Initialize Alembic**:
```bash
alembic init alembic
```

**Configure Alembic**:
```python
# alembic/env.py
from logging.config import fileConfig
from sqlalchemy import engine_from_config, pool
from alembic import context

from app.models.base import Base
from app.config import settings

# Import all models
from app.models import *

config = context.config
config.set_main_option("sqlalchemy.url", settings.DATABASE_URL)

target_metadata = Base.metadata

def run_migrations_online():
    connectable = engine_from_config(
        config.get_section(config.config_ini_section),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )

    with connectable.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)
        with context.begin_transaction():
            context.run_migrations()
```

**Create Initial Migration**:
```bash
alembic revision --autogenerate -m "Initial schema"
```

**Migration Versions** (matching Go migration history):
```
alembic/versions/
├── 001_initial_schema.py          # Equivalent to v1.0.0
├── 002_add_queue_system.py        # Equivalent to v6.0.0
├── 003_add_azure_integration.py   # Equivalent to v7.0.0
├── 004_add_shopify.py              # Equivalent to v7.1.0
└── ...
```

### 3.4 Query Migration

**Queries.sql → SQLAlchemy Queries**

Current: Named SQL queries in `queries.sql` (1,809 lines)
Target: SQLAlchemy query methods in core modules

**Example Migration**:

**Current (Go + goyesql)**:
```sql
-- name: get-subscriber
SELECT * FROM subscribers WHERE id = $1;

-- name: query-subscribers
SELECT s.*,
    COALESCE(l.lists, '[]') AS lists
FROM subscribers s
LEFT JOIN LATERAL (
    SELECT COALESCE(JSON_AGG(
        JSON_BUILD_OBJECT('id', l.id, 'name', l.name)
    ), '[]') AS lists
    FROM lists l
    WHERE l.id = ANY(
        SELECT list_id FROM subscriber_lists WHERE subscriber_id = s.id
    )
) l ON true
WHERE %query%
ORDER BY %order% LIMIT $1 OFFSET $2;
```

**Target (Python + SQLAlchemy)**:
```python
# app/core/subscribers.py
from sqlalchemy import select, func
from sqlalchemy.orm import selectinload

async def get_subscriber(db: AsyncSession, subscriber_id: int) -> Subscriber:
    result = await db.execute(
        select(Subscriber)
        .where(Subscriber.id == subscriber_id)
    )
    return result.scalar_one_or_none()

async def query_subscribers(
    db: AsyncSession,
    where_clause=None,
    order_by=None,
    limit: int = 100,
    offset: int = 0
):
    query = select(Subscriber).options(
        selectinload(Subscriber.subscriptions).selectinload(SubscriberList.list)
    )

    if where_clause is not None:
        query = query.where(where_clause)

    if order_by is not None:
        query = query.order_by(order_by)

    query = query.limit(limit).offset(offset)

    result = await db.execute(query)
    return result.scalars().all()
```

**Complex Queries** (require careful porting):
- create-campaign (with CTEs, JSON aggregation)
- next-campaign-subscribers (batch fetching with chunking)
- record-bounce (complex conditional logic)
- queue-campaign-emails (bulk insert with Smart Sending check)

### 3.5 Data Migration Script

**For migrating from existing listmonk installation**:

```python
# scripts/migrate_data.py
import asyncio
import asyncpg
from sqlalchemy.ext.asyncio import create_async_engine

async def migrate_data():
    # Source: Current listmonk database
    source_conn = await asyncpg.connect(
        host='listmonk420-db.postgres.database.azure.com',
        user='listmonkadmin',
        password='T@intshr3dd3r',
        database='listmonk'
    )

    # Target: New Python app database
    target_engine = create_async_engine("postgresql+asyncpg://...")

    # Migrate tables in order (respecting foreign keys)
    tables = [
        'subscribers',
        'lists',
        'subscriber_lists',
        'templates',
        'campaigns',
        'campaign_lists',
        'media',
        'campaign_media',
        'links',
        'campaign_views',
        'link_clicks',
        'bounces',
        'users',
        'roles',
        'email_queue',
        # ... all tables
    ]

    for table in tables:
        print(f"Migrating {table}...")
        rows = await source_conn.fetch(f"SELECT * FROM {table}")

        # Bulk insert into target
        async with target_engine.begin() as conn:
            # Insert rows
            pass

    await source_conn.close()

if __name__ == "__main__":
    asyncio.run(migrate_data())
```

---

## 4. BACKEND PORTING PLAN

### 4.1 API Endpoint Mapping Overview

**Total Endpoints**: 156+

**Categories**:
- Campaigns: 20+ endpoints
- Queue: 10+ endpoints
- Subscribers: 15+ endpoints
- Lists: 5+ endpoints
- Templates: 6+ endpoints
- Media: 3+ endpoints
- Bounces: 5+ endpoints
- Webhook Logs: 4+ endpoints
- Webhooks (external): 7+ endpoints
- Users & Auth: 15+ endpoints
- Roles: 6+ endpoints
- Settings: 5+ endpoints
- Import: 4+ endpoints
- Maintenance: 4+ endpoints
- Dashboard: 2+ endpoints
- Public: 12+ endpoints

### 4.2 Campaign API Endpoints

**Go Echo → FastAPI Mapping**:

```python
# app/api/v1/campaigns.py
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession
from typing import List, Optional

from app.dependencies import get_db, get_current_user
from app.models.user import User
from app.schemas.campaign import Campaign, CampaignCreate, CampaignUpdate
from app.core import campaigns as campaign_core

router = APIRouter()

@router.get("/", response_model=List[Campaign])
async def get_campaigns(
    page: int = Query(1, ge=1),
    per_page: int = Query(20, ge=1, le=100),
    status: Optional[str] = None,
    query: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get list of campaigns with pagination and filtering."""
    campaigns = await campaign_core.get_campaigns(
        db,
        page=page,
        per_page=per_page,
        status=status,
        search_query=query
    )
    return campaigns

@router.get("/{campaign_id}", response_model=Campaign)
async def get_campaign(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get single campaign by ID."""
    campaign = await campaign_core.get_campaign(db, campaign_id)
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return campaign

@router.post("/", response_model=Campaign, status_code=201)
async def create_campaign(
    campaign: CampaignCreate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Create a new campaign."""
    # Permission check
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    return await campaign_core.create_campaign(db, campaign)

@router.put("/{campaign_id}", response_model=Campaign)
async def update_campaign(
    campaign_id: int,
    campaign: CampaignUpdate,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Update an existing campaign."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    updated = await campaign_core.update_campaign(db, campaign_id, campaign)
    if not updated:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return updated

@router.delete("/{campaign_id}", status_code=204)
async def delete_campaign(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Delete a campaign."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    success = await campaign_core.delete_campaign(db, campaign_id)
    if not success:
        raise HTTPException(status_code=404, detail="Campaign not found")

@router.put("/{campaign_id}/status")
async def update_campaign_status(
    campaign_id: int,
    status: str,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Change campaign status (draft, running, paused, etc.)"""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    campaign = await campaign_core.update_campaign_status(db, campaign_id, status)
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return campaign

@router.post("/{campaign_id}/test")
async def test_campaign(
    campaign_id: int,
    emails: List[str],
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Send test emails for a campaign."""
    campaign = await campaign_core.get_campaign(db, campaign_id)
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")

    # Send test emails
    await campaign_core.send_test_emails(db, campaign, emails)
    return {"status": "sent"}

@router.get("/{campaign_id}/preview")
async def preview_campaign(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Preview campaign HTML."""
    html = await campaign_core.render_campaign_preview(db, campaign_id)
    return {"html": html}

@router.get("/analytics/views")
async def get_campaign_views(
    campaign_id: Optional[int] = None,
    from_date: Optional[str] = None,
    to_date: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get campaign view analytics."""
    return await campaign_core.get_view_analytics(
        db, campaign_id, from_date, to_date
    )

@router.get("/analytics/clicks")
async def get_campaign_clicks(
    campaign_id: Optional[int] = None,
    from_date: Optional[str] = None,
    to_date: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get campaign click analytics."""
    return await campaign_core.get_click_analytics(
        db, campaign_id, from_date, to_date
    )

@router.get("/{campaign_id}/azure-analytics")
async def get_azure_analytics(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get Azure Event Grid analytics for campaign."""
    return await campaign_core.get_azure_analytics(db, campaign_id)

@router.get("/{campaign_id}/purchases/stats")
async def get_purchase_stats(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get Shopify purchase attribution stats for campaign."""
    return await campaign_core.get_purchase_stats(db, campaign_id)

@router.post("/{campaign_id}/remove-sent-today")
async def remove_sent_subscribers(
    campaign_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Remove sent subscribers from campaign lists."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    count = await campaign_core.remove_sent_subscribers(db, campaign_id)
    return {"removed": count}

# ... more campaign endpoints
```

**Business Logic**:
```python
# app/core/campaigns.py
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update, delete
from sqlalchemy.orm import selectinload
from typing import List, Optional

from app.models.campaign import Campaign
from app.models.subscriber import Subscriber
from app.schemas.campaign import CampaignCreate, CampaignUpdate

async def get_campaigns(
    db: AsyncSession,
    page: int = 1,
    per_page: int = 20,
    status: Optional[str] = None,
    search_query: Optional[str] = None
) -> List[Campaign]:
    """Get campaigns with pagination and filtering."""
    query = select(Campaign).options(
        selectinload(Campaign.lists),
        selectinload(Campaign.template)
    )

    if status:
        query = query.where(Campaign.status == status)

    if search_query:
        query = query.where(Campaign.name.ilike(f"%{search_query}%"))

    query = query.order_by(Campaign.created_at.desc())
    query = query.limit(per_page).offset((page - 1) * per_page)

    result = await db.execute(query)
    return result.scalars().all()

async def create_campaign(
    db: AsyncSession,
    campaign: CampaignCreate
) -> Campaign:
    """Create a new campaign."""
    # Create campaign object
    db_campaign = Campaign(**campaign.dict(exclude={'list_ids', 'media_ids'}))
    db.add(db_campaign)
    await db.flush()

    # Add campaign-list associations
    if campaign.list_ids:
        for list_id in campaign.list_ids:
            campaign_list = CampaignList(
                campaign_id=db_campaign.id,
                list_id=list_id
            )
            db.add(campaign_list)

    # Add media associations
    if campaign.media_ids:
        for media_id in campaign.media_ids:
            campaign_media = CampaignMedia(
                campaign_id=db_campaign.id,
                media_id=media_id
            )
            db.add(campaign_media)

    await db.commit()
    await db.refresh(db_campaign)

    return db_campaign

async def update_campaign_status(
    db: AsyncSession,
    campaign_id: int,
    new_status: str
) -> Optional[Campaign]:
    """Update campaign status and trigger appropriate actions."""
    campaign = await get_campaign(db, campaign_id)
    if not campaign:
        return None

    old_status = campaign.status
    campaign.status = new_status

    # Handle status transitions
    if new_status == "running" and old_status != "running":
        # Check if queue-based
        if campaign.messenger == "automatic":
            # Queue all emails
            await queue_campaign_emails(db, campaign)
        else:
            # Mark for campaign processor
            campaign.started_at = datetime.utcnow()

    elif new_status == "paused":
        # Pause campaign
        pass

    elif new_status == "cancelled":
        # Cancel queued emails
        if campaign.use_queue:
            await cancel_campaign_queue(db, campaign_id)

    await db.commit()
    await db.refresh(campaign)

    return campaign

async def send_test_emails(
    db: AsyncSession,
    campaign: Campaign,
    emails: List[str]
):
    """Send test emails for a campaign."""
    from app.manager.campaign_manager import CampaignManager

    manager = CampaignManager(db)

    for email in emails:
        # Create temporary subscriber
        subscriber = Subscriber(
            email=email,
            name="Test Subscriber"
        )

        # Render and send
        await manager.send_message(campaign, subscriber, is_test=True)

async def render_campaign_preview(
    db: AsyncSession,
    campaign_id: int
) -> str:
    """Render campaign HTML for preview."""
    campaign = await get_campaign(db, campaign_id)
    if not campaign:
        return ""

    from app.manager.message import render_campaign_html

    # Use dummy subscriber for preview
    dummy_subscriber = Subscriber(
        email="preview@example.com",
        name="Preview Subscriber",
        attribs={}
    )

    html = await render_campaign_html(campaign, dummy_subscriber)
    return html

# ... more core functions
```

### 4.3 Queue API Endpoints

```python
# app/api/v1/queue.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.dependencies import get_db, get_current_user
from app.core import queue as queue_core

router = APIRouter()

@router.get("/items")
async def get_queue_items(
    page: int = 1,
    per_page: int = 100,
    status: Optional[str] = None,
    campaign_id: Optional[int] = None,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get queue items with pagination."""
    return await queue_core.get_queue_items(
        db, page, per_page, status, campaign_id
    )

@router.get("/stats")
async def get_queue_stats(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get queue statistics."""
    return await queue_core.get_queue_stats(db)

@router.get("/servers")
async def get_server_capacity(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Get SMTP server capacity information."""
    return await queue_core.get_server_capacity(db)

@router.put("/{item_id}/cancel")
async def cancel_queue_item(
    item_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Cancel a single queue item."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403)

    await queue_core.cancel_queue_item(db, item_id)
    return {"status": "cancelled"}

@router.put("/{item_id}/retry")
async def retry_queue_item(
    item_id: int,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Retry a failed queue item."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403)

    await queue_core.retry_queue_item(db, item_id)
    return {"status": "queued"}

@router.post("/clear")
async def clear_queue(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Clear all queued emails."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403)

    count = await queue_core.clear_all_queued_emails(db)
    return {"cleared": count}

@router.put("/pause")
async def toggle_queue_pause(
    paused: bool,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Pause or resume queue processing."""
    if not current_user.has_permission("settings:manage"):
        raise HTTPException(status_code=403)

    await queue_core.set_queue_paused(db, paused)
    return {"paused": paused}

@router.post("/send-all")
async def send_all_queued(
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user)
):
    """Send all queued emails immediately (bypasses time windows)."""
    if not current_user.has_permission("campaigns:manage"):
        raise HTTPException(status_code=403)

    count = await queue_core.send_all_queued_emails(db)
    return {"sent": count}
```

### 4.4 Campaign Manager (Email Sending)

```python
# app/manager/campaign_manager.py
import asyncio
from datetime import datetime
from typing import List, Optional

from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update

from app.models.campaign import Campaign
from app.models.subscriber import Subscriber
from app.messenger.base import BaseMessenger
from app.messenger.email import EmailMessenger
from app.manager.message import CampaignMessage

class CampaignManager:
    def __init__(self, db: AsyncSession):
        self.db = db
        self.messengers: dict[str, BaseMessenger] = {}
        self.batch_size = 1000
        self.concurrency = 10
        self.message_rate = 10  # messages per second

        # Initialize messengers
        self._init_messengers()

    def _init_messengers(self):
        """Initialize email messengers from settings."""
        # Load SMTP servers from settings
        smtp_servers = await self._get_smtp_servers()

        for smtp in smtp_servers:
            if smtp.get('name'):
                # Named server -> dedicated messenger
                messenger_name = f"email-{smtp['name']}"
                self.messengers[messenger_name] = EmailMessenger(smtp)
            else:
                # Unnamed -> add to default pool
                if 'email' not in self.messengers:
                    self.messengers['email'] = EmailMessenger(smtp, is_pool=True)
                else:
                    self.messengers['email'].add_server(smtp)

    async def scan_campaigns(self):
        """Scan for campaigns that need processing."""
        # Get running campaigns (non-queue-based)
        query = select(Campaign).where(
            Campaign.status == 'running',
            Campaign.use_queue == False
        )

        result = await self.db.execute(query)
        campaigns = result.scalars().all()

        for campaign in campaigns:
            asyncio.create_task(self.process_campaign(campaign))

    async def process_campaign(self, campaign: Campaign):
        """Process a single campaign."""
        try:
            while campaign.sent < campaign.to_send:
                # Fetch next batch of subscribers
                subscribers = await self._fetch_next_batch(campaign)

                if not subscribers:
                    # Campaign complete
                    await self._finish_campaign(campaign)
                    break

                # Send emails concurrently
                await self._send_batch(campaign, subscribers)

                # Update campaign counters
                campaign.sent += len(subscribers)
                await self.db.commit()

                # Rate limiting
                await asyncio.sleep(len(subscribers) / self.message_rate)

        except Exception as e:
            # Mark campaign as failed
            campaign.status = 'cancelled'
            await self.db.commit()
            raise

    async def _fetch_next_batch(
        self,
        campaign: Campaign
    ) -> List[Subscriber]:
        """Fetch next batch of subscribers for campaign."""
        # Complex query to get subscribers from campaign lists
        # with status filtering, already-sent exclusion, etc.
        # (Similar to Go next-campaign-subscribers query)
        pass

    async def _send_batch(
        self,
        campaign: Campaign,
        subscribers: List[Subscriber]
    ):
        """Send emails to a batch of subscribers."""
        semaphore = asyncio.Semaphore(self.concurrency)

        async def send_one(subscriber: Subscriber):
            async with semaphore:
                await self.send_message(campaign, subscriber)

        tasks = [send_one(sub) for sub in subscribers]
        await asyncio.gather(*tasks, return_exceptions=True)

    async def send_message(
        self,
        campaign: Campaign,
        subscriber: Subscriber,
        is_test: bool = False
    ):
        """Send a single campaign message."""
        # Get messenger
        messenger = self.messengers.get(campaign.messenger)
        if not messenger:
            raise ValueError(f"Messenger not found: {campaign.messenger}")

        # Render message
        message = await CampaignMessage.render(
            campaign,
            subscriber,
            self.db
        )

        # Send via messenger
        await messenger.push(message)

        if not is_test:
            # Record send
            # Update campaign_views, etc.
            pass

    async def _finish_campaign(self, campaign: Campaign):
        """Mark campaign as finished."""
        campaign.status = 'finished'
        campaign.queue_completed_at = datetime.utcnow()
        await self.db.commit()
```

### 4.5 Message Rendering (Jinja2)

```python
# app/manager/message.py
from jinja2 import Environment, FileSystemLoader
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from typing import Dict, Any

from app.models.campaign import Campaign
from app.models.subscriber import Subscriber
from app.models.template import Template

class CampaignMessage:
    def __init__(
        self,
        to_email: str,
        from_email: str,
        subject: str,
        html_body: str,
        text_body: str,
        headers: Dict[str, str] = None
    ):
        self.to_email = to_email
        self.from_email = from_email
        self.subject = subject
        self.html_body = html_body
        self.text_body = text_body
        self.headers = headers or {}

    @classmethod
    async def render(
        cls,
        campaign: Campaign,
        subscriber: Subscriber,
        db: AsyncSession
    ) -> "CampaignMessage":
        """Render a campaign message for a subscriber."""
        # Load template
        template = await db.get(Template, campaign.template_id)

        # Prepare template context
        context = {
            'campaign': campaign,
            'subscriber': subscriber,
            'TrackLink': lambda url: track_link(url, campaign, subscriber),
            'UnsubscribeURL': lambda: unsubscribe_url(campaign, subscriber),
            'TrackView': lambda: view_pixel_url(campaign, subscriber),
            # ... more template functions
        }

        # Render subject
        subject_template = Environment().from_string(campaign.subject)
        subject = subject_template.render(**context)

        # Render body
        if campaign.content_type == 'html':
            html_body = render_html_body(campaign, template, context)
            text_body = html_to_text(html_body)
        elif campaign.content_type == 'markdown':
            html_body = markdown_to_html(campaign.body)
            html_body = render_html_body_with_template(html_body, template, context)
            text_body = html_to_text(html_body)
        elif campaign.content_type == 'plain':
            text_body = campaign.body
            html_body = f"<pre>{text_body}</pre>"
        else:  # richtext
            html_body = render_html_body(campaign, template, context)
            text_body = campaign.altbody or html_to_text(html_body)

        return cls(
            to_email=subscriber.email,
            from_email=campaign.from_email,
            subject=subject,
            html_body=html_body,
            text_body=text_body,
            headers=campaign.headers
        )

    def to_mime(self) -> MIMEMultipart:
        """Convert to MIME message for SMTP sending."""
        msg = MIMEMultipart('alternative')
        msg['To'] = self.to_email
        msg['From'] = self.from_email
        msg['Subject'] = self.subject

        # Add custom headers
        for key, value in self.headers.items():
            msg[key] = value

        # Add text and HTML parts
        msg.attach(MIMEText(self.text_body, 'plain'))
        msg.attach(MIMEText(self.html_body, 'html'))

        return msg

def render_html_body(
    campaign: Campaign,
    template: Template,
    context: Dict[str, Any]
) -> str:
    """Render campaign body with template wrapper."""
    env = Environment(loader=FileSystemLoader('app/templates/email'))

    # Render campaign body first
    body_template = env.from_string(campaign.body)
    rendered_body = body_template.render(**context)

    # Wrap with template
    if template:
        template_obj = env.from_string(template.body)
        context['Body'] = rendered_body
        html = template_obj.render(**context)
    else:
        html = rendered_body

    return html

def track_link(url: str, campaign: Campaign, subscriber: Subscriber) -> str:
    """Generate tracked link URL."""
    # Create/get link UUID
    # Return: https://listmonk.app/link/{link_uuid}/{campaign_uuid}/{subscriber_uuid}
    pass

def unsubscribe_url(campaign: Campaign, subscriber: Subscriber) -> str:
    """Generate unsubscribe URL."""
    return f"https://listmonk.app/subscription/{campaign.uuid}/{subscriber.uuid}"

def view_pixel_url(campaign: Campaign, subscriber: Subscriber) -> str:
    """Generate view tracking pixel URL."""
    return f"https://listmonk.app/campaign/{campaign.uuid}/{subscriber.uuid}/px.png"
```

### 4.6 SMTP Messenger (Async)

```python
# app/messenger/email.py
import aiosmtplib
from email.mime.multipart import MIMEMultipart
from typing import Optional, List

from app.messenger.base import BaseMessenger
from app.manager.message import CampaignMessage

class EmailMessenger(BaseMessenger):
    def __init__(self, config: dict, is_pool: bool = False):
        self.config = config
        self.is_pool = is_pool
        self.servers = [config] if not is_pool else []
        self.max_conns = config.get('max_conns', 10)
        self.daily_limit = config.get('daily_limit', 0)
        self.from_email = config.get('from_email', '')
        self.name = config.get('name', 'email')

        # Connection pool
        self.connections = asyncio.Queue(maxsize=self.max_conns)
        self._init_connections()

    def _init_connections(self):
        """Initialize SMTP connection pool."""
        for _ in range(self.max_conns):
            self.connections.put_nowait(None)

    async def _get_connection(self) -> aiosmtplib.SMTP:
        """Get an SMTP connection from pool."""
        conn = await self.connections.get()

        if conn is None or not conn.is_connected:
            # Create new connection
            conn = aiosmtplib.SMTP(
                hostname=self.config['host'],
                port=self.config['port'],
                use_tls=self.config.get('tls_type') == 'TLS',
                start_tls=self.config.get('tls_type') == 'STARTTLS',
                timeout=self.config.get('wait_timeout', 5)
            )

            await conn.connect()

            # Authenticate
            if self.config.get('auth_protocol') != 'none':
                await conn.login(
                    self.config['username'],
                    self.config['password']
                )

        return conn

    async def _return_connection(self, conn: aiosmtplib.SMTP):
        """Return connection to pool."""
        await self.connections.put(conn)

    async def push(self, message: CampaignMessage):
        """Send a message via SMTP."""
        conn = await self._get_connection()

        try:
            # Convert to MIME
            mime_msg = message.to_mime()

            # Send
            await conn.send_message(mime_msg)

        finally:
            await self._return_connection(conn)

    async def flush(self):
        """Flush any pending messages."""
        pass

    async def close(self):
        """Close all connections."""
        while not self.connections.empty():
            conn = await self.connections.get()
            if conn and conn.is_connected:
                await conn.quit()
```

### 4.7 Bounce Handling

**POP3 Mailbox Scanner**:
```python
# app/bounce/mailbox.py
import asyncio
from poplib import POP3_SSL
from email import message_from_bytes
from email.utils import parseaddr

async def scan_bounce_mailbox(mailbox_config: dict, record_bounce_callback):
    """Scan a POP3 mailbox for bounce messages."""
    while True:
        try:
            # Connect to POP3 server
            pop3 = POP3_SSL(
                mailbox_config['host'],
                mailbox_config['port']
            )

            pop3.user(mailbox_config['username'])
            pop3.pass_(mailbox_config['password'])

            # Get message count
            num_messages = len(pop3.list()[1])

            for i in range(1, num_messages + 1):
                # Fetch message
                response, lines, octets = pop3.retr(i)
                msg_data = b'\n'.join(lines)
                msg = message_from_bytes(msg_data)

                # Parse bounce
                bounce_email = extract_bounce_email(msg)
                bounce_type = determine_bounce_type(msg)

                if bounce_email:
                    # Record bounce
                    await record_bounce_callback(
                        email=bounce_email,
                        bounce_type=bounce_type,
                        source='mailbox',
                        meta={'headers': dict(msg.items())}
                    )

                # Delete processed message
                pop3.dele(i)

            pop3.quit()

        except Exception as e:
            print(f"Error scanning mailbox: {e}")

        # Sleep until next scan
        await asyncio.sleep(mailbox_config['scan_interval'] * 60)

def extract_bounce_email(msg) -> Optional[str]:
    """Extract bounced email address from bounce message."""
    # Parse Return-Path, Original-Recipient, etc.
    # Complex parsing logic
    pass

def determine_bounce_type(msg) -> str:
    """Determine bounce type (hard, soft, complaint)."""
    # Analyze bounce message content
    # Return 'hard', 'soft', or 'complaint'
    pass
```

**Bounce Webhooks**:
```python
# app/bounce/webhooks/ses.py
from fastapi import Request

async def handle_ses_bounce(request: Request, record_bounce_callback):
    """Handle Amazon SES bounce webhook."""
    data = await request.json()

    # Parse SNS message
    if data.get('Type') == 'SubscriptionConfirmation':
        # Confirm subscription
        pass

    elif data.get('Type') == 'Notification':
        message = json.loads(data['Message'])

        if message.get('notificationType') == 'Bounce':
            # Process bounce
            bounce_type = 'hard' if message['bounce']['bounceType'] == 'Permanent' else 'soft'

            for recipient in message['bounce']['bouncedRecipients']:
                await record_bounce_callback(
                    email=recipient['emailAddress'],
                    bounce_type=bounce_type,
                    source='ses',
                    meta=message
                )
```

---

## 5. FRONTEND PORTING PLAN

### 5.1 React Application Structure

**Technology Stack**:
- React 18.2+
- TypeScript 5.0+
- Vite (build tool)
- React Router v6
- Zustand or Redux Toolkit (state management)
- Material-UI v5 (UI library - alternative: Ant Design)
- React Hook Form (forms)
- TanStack Query (data fetching/caching)
- TinyMCE React (rich text editor)
- CodeMirror 6 (code editor)
- Chart.js with react-chartjs-2

### 5.2 Component Mapping (Vue 2 → React)

**Page Components** (21 pages):

| Vue 2 Component | React Component | Notes |
|----------------|-----------------|-------|
| Dashboard.vue | Dashboard.tsx | Convert Vuex to Zustand, Chart.js integration |
| Campaigns.vue | Campaigns/CampaignList.tsx | Material-UI DataGrid or Ant Table |
| Campaign.vue | Campaigns/CampaignEditor.tsx | Complex form with React Hook Form |
| CampaignAnalytics.vue | Campaigns/CampaignAnalytics.tsx | Chart components, date range picker |
| Queue.vue | Campaigns/Queue.tsx | Real-time updates via polling or WebSocket |
| Subscribers.vue | Subscribers/SubscriberList.tsx | DataGrid with bulk actions |
| SubscriberForm.vue | Subscribers/SubscriberForm.tsx | React Hook Form with validation |
| Lists.vue | Lists/ListList.tsx | Simple list with CRUD |
| ListForm.vue | Lists/ListForm.tsx | Form component |
| Templates.vue | Templates/TemplateList.tsx | Template library |
| TemplateForm.vue | Templates/TemplateForm.tsx | Editor integration |
| Media.vue | Media/MediaLibrary.tsx | Upload, grid view |
| Import.vue | Subscribers/Import.tsx | File upload, progress tracking |
| Bounces.vue | Bounces/BounceList.tsx | DataGrid with filtering |
| Settings.vue | Settings/SettingsLayout.tsx | Tab container |
| Users.vue | Users/UserList.tsx | User management |
| Roles.vue | Users/RoleList.tsx | RBAC management |
| Logs.vue | Settings/Logs.tsx | Log viewer |
| WebhookLogs.vue | Settings/WebhookLogs.tsx | Webhook log viewer |
| Maintenance.vue | Settings/Maintenance.tsx | Maintenance tools |
| Login.vue | Auth/Login.tsx | Login form |

**Reusable Components** (13 components):

| Vue 2 Component | React Component | Library/Implementation |
|----------------|-----------------|------------------------|
| Editor.vue | Editors/CampaignEditor.tsx | TinyMCE + CodeMirror wrapper |
| RichtextEditor.vue | Editors/RichTextEditor.tsx | TinyMCE React |
| VisualEditor.vue | Editors/VisualEditor.tsx | Iframe integration |
| CodeEditor.vue | Editors/CodeEditor.tsx | CodeMirror 6 React |
| ListSelector.vue | Forms/ListSelector.tsx | Multi-select with Material-UI |
| CopyText.vue | Common/CopyText.tsx | Copy-to-clipboard button |
| Chart.vue | Charts/Chart.tsx | Chart.js wrapper |
| BarChart.vue | Charts/BarChart.tsx | react-chartjs-2 |
| CampaignPreview.vue | Campaigns/CampaignPreview.tsx | Modal with iframe |
| CampaignAzureAnalytics.vue | Campaigns/AzureAnalytics.tsx | Azure metrics display |
| Navigation.vue | Layout/Navigation.tsx | Material-UI drawer/sidebar |
| LogView.vue | Common/LogView.tsx | Virtual scrolling log viewer |
| EmptyPlaceholder.vue | Common/EmptyState.tsx | Empty state component |

### 5.3 State Management Migration

**Vuex → Zustand Pattern**:

**Vue 2 Vuex Store**:
```javascript
// Vue 2 store/index.js
export default new Vuex.Store({
  state: {
    campaigns: [],
    loading: {
      campaigns: false
    }
  },
  mutations: {
    SET_CAMPAIGNS(state, campaigns) {
      state.campaigns = campaigns;
    },
    SET_LOADING(state, { model, status }) {
      state.loading[model] = status;
    }
  },
  actions: {
    async fetchCampaigns({ commit }) {
      commit('SET_LOADING', { model: 'campaigns', status: true });
      const campaigns = await api.getCampaigns();
      commit('SET_CAMPAIGNS', campaigns);
      commit('SET_LOADING', { model: 'campaigns', status: false });
    }
  }
});
```

**React Zustand Store**:
```typescript
// src/stores/campaignStore.ts
import create from 'zustand';
import { devtools } from 'zustand/middleware';
import { Campaign, CampaignCreate, CampaignUpdate } from '@/types/campaign';
import * as campaignApi from '@/api/campaigns';

interface CampaignState {
  campaigns: Campaign[];
  selectedCampaign: Campaign | null;
  loading: boolean;
  error: string | null;

  // Actions
  fetchCampaigns: (filters?: CampaignFilters) => Promise<void>;
  fetchCampaign: (id: number) => Promise<void>;
  createCampaign: (data: CampaignCreate) => Promise<Campaign>;
  updateCampaign: (id: number, data: CampaignUpdate) => Promise<Campaign>;
  deleteCampaign: (id: number) => Promise<void>;
  updateCampaignStatus: (id: number, status: string) => Promise<void>;
  clearError: () => void;
}

export const useCampaignStore = create<CampaignState>()(
  devtools((set, get) => ({
    campaigns: [],
    selectedCampaign: null,
    loading: false,
    error: null,

    fetchCampaigns: async (filters) => {
      set({ loading: true, error: null });
      try {
        const campaigns = await campaignApi.getCampaigns(filters);
        set({ campaigns, loading: false });
      } catch (error) {
        set({ error: error.message, loading: false });
      }
    },

    fetchCampaign: async (id) => {
      set({ loading: true, error: null });
      try {
        const campaign = await campaignApi.getCampaign(id);
        set({ selectedCampaign: campaign, loading: false });
      } catch (error) {
        set({ error: error.message, loading: false });
      }
    },

    createCampaign: async (data) => {
      set({ loading: true, error: null });
      try {
        const campaign = await campaignApi.createCampaign(data);
        set((state) => ({
          campaigns: [campaign, ...state.campaigns],
          loading: false
        }));
        return campaign;
      } catch (error) {
        set({ error: error.message, loading: false });
        throw error;
      }
    },

    updateCampaign: async (id, data) => {
      set({ loading: true, error: null });
      try {
        const updated = await campaignApi.updateCampaign(id, data);
        set((state) => ({
          campaigns: state.campaigns.map((c) =>
            c.id === id ? updated : c
          ),
          selectedCampaign: state.selectedCampaign?.id === id ? updated : state.selectedCampaign,
          loading: false
        }));
        return updated;
      } catch (error) {
        set({ error: error.message, loading: false });
        throw error;
      }
    },

    deleteCampaign: async (id) => {
      set({ loading: true, error: null });
      try {
        await campaignApi.deleteCampaign(id);
        set((state) => ({
          campaigns: state.campaigns.filter((c) => c.id !== id),
          loading: false
        }));
      } catch (error) {
        set({ error: error.message, loading: false });
        throw error;
      }
    },

    updateCampaignStatus: async (id, status) => {
      try {
        const updated = await campaignApi.updateCampaignStatus(id, status);
        set((state) => ({
          campaigns: state.campaigns.map((c) =>
            c.id === id ? { ...c, status: updated.status } : c
          )
        }));
      } catch (error) {
        set({ error: error.message });
        throw error;
      }
    },

    clearError: () => set({ error: null })
  }), { name: 'CampaignStore' })
);
```

**Using TanStack Query** (alternative/complementary to Zustand):
```typescript
// src/hooks/useCampaigns.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as campaignApi from '@/api/campaigns';

export function useCampaigns(filters?) {
  return useQuery({
    queryKey: ['campaigns', filters],
    queryFn: () => campaignApi.getCampaigns(filters)
  });
}

export function useCampaign(id: number) {
  return useQuery({
    queryKey: ['campaign', id],
    queryFn: () => campaignApi.getCampaign(id),
    enabled: !!id
  });
}

export function useCreateCampaign() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: campaignApi.createCampaign,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['campaigns'] });
    }
  });
}

export function useUpdateCampaign() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: CampaignUpdate }) =>
      campaignApi.updateCampaign(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['campaign', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['campaigns'] });
    }
  });
}

// Usage in component:
function CampaignList() {
  const { data: campaigns, isLoading, error } = useCampaigns();
  const createMutation = useCreateCampaign();

  const handleCreate = async (data: CampaignCreate) => {
    await createMutation.mutateAsync(data);
  };

  // ...
}
```

### 5.4 API Client Implementation

```typescript
// src/api/client.ts
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import { camelizeKeys, decamelizeKeys } from 'humps';

const baseURL = import.meta.env.VITE_API_URL || '/api';

export const apiClient: AxiosInstance = axios.create({
  baseURL,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Request interceptor - transform to snake_case
apiClient.interceptors.request.use(
  (config) => {
    if (config.data) {
      config.data = decamelizeKeys(config.data);
    }
    if (config.params) {
      config.params = decamelizeKeys(config.params);
    }

    // Add auth token if exists
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor - transform to camelCase
apiClient.interceptors.response.use(
  (response) => {
    if (response.data && typeof response.data === 'object') {
      response.data = camelizeKeys(response.data);
    }
    return response;
  },
  (error) => {
    // Handle errors globally
    if (error.response?.status === 401) {
      // Redirect to login
      window.location.href = '/login';
    }

    return Promise.reject(error);
  }
);
```

```typescript
// src/api/campaigns.ts
import { apiClient } from './client';
import {
  Campaign,
  CampaignCreate,
  CampaignUpdate,
  CampaignFilters
} from '@/types/campaign';

export async function getCampaigns(filters?: CampaignFilters): Promise<Campaign[]> {
  const { data } = await apiClient.get('/v1/campaigns', { params: filters });
  return data;
}

export async function getCampaign(id: number): Promise<Campaign> {
  const { data } = await apiClient.get(`/v1/campaigns/${id}`);
  return data;
}

export async function createCampaign(campaign: CampaignCreate): Promise<Campaign> {
  const { data } = await apiClient.post('/v1/campaigns', campaign);
  return data;
}

export async function updateCampaign(
  id: number,
  campaign: CampaignUpdate
): Promise<Campaign> {
  const { data} = await apiClient.put(`/v1/campaigns/${id}`, campaign);
  return data;
}

export async function deleteCampaign(id: number): Promise<void> {
  await apiClient.delete(`/v1/campaigns/${id}`);
}

export async function updateCampaignStatus(
  id: number,
  status: string
): Promise<Campaign> {
  const { data } = await apiClient.put(`/v1/campaigns/${id}/status`, { status });
  return data;
}

export async function testCampaign(
  id: number,
  emails: string[]
): Promise<void> {
  await apiClient.post(`/v1/campaigns/${id}/test`, { emails });
}

export async function getCampaignAnalytics(
  id: number,
  type: 'views' | 'clicks' | 'bounces'
): Promise<any> {
  const { data } = await apiClient.get(`/v1/campaigns/analytics/${type}`, {
    params: { campaign_id: id }
  });
  return data;
}

// ... more campaign API methods
```

### 5.5 Example React Component

```typescript
// src/pages/Campaigns/CampaignList.tsx
import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Menu,
  MenuItem
} from '@mui/material';
import {
  Add as AddIcon,
  MoreVert as MoreIcon,
  PlayArrow as PlayIcon,
  Pause as PauseIcon
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useCampaignStore } from '@/stores/campaignStore';
import { Campaign } from '@/types/campaign';

export const CampaignList: React.FC = () => {
  const navigate = useNavigate();
  const { campaigns, loading, fetchCampaigns, updateCampaignStatus, deleteCampaign } =
    useCampaignStore();

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [selectedCampaign, setSelectedCampaign] = useState<Campaign | null>(null);

  useEffect(() => {
    fetchCampaigns();
  }, []);

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, campaign: Campaign) => {
    setAnchorEl(event.currentTarget);
    setSelectedCampaign(campaign);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
    setSelectedCampaign(null);
  };

  const handleStatusChange = async (status: string) => {
    if (selectedCampaign) {
      await updateCampaignStatus(selectedCampaign.id, status);
      handleMenuClose();
    }
  };

  const handleDelete = async () => {
    if (selectedCampaign && confirm('Delete campaign?')) {
      await deleteCampaign(selectedCampaign.id);
      handleMenuClose();
    }
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
      draft: 'default',
      running: 'success',
      scheduled: 'info',
      paused: 'warning',
      cancelled: 'error',
      finished: 'default'
    };
    return colors[status] || 'default';
  };

  if (loading) return <div>Loading...</div>;

  return (
    <Box p={3}>
      <Box display="flex" justifyContent="space-between" mb={3}>
        <h1>Campaigns</h1>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => navigate('/campaigns/new')}
        >
          New Campaign
        </Button>
      </Box>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Lists</TableCell>
              <TableCell>Sent</TableCell>
              <TableCell>Views</TableCell>
              <TableCell>Clicks</TableCell>
              <TableCell>Created</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {campaigns.map((campaign) => (
              <TableRow
                key={campaign.id}
                hover
                onClick={() => navigate(`/campaigns/${campaign.id}`)}
                style={{ cursor: 'pointer' }}
              >
                <TableCell>{campaign.name}</TableCell>
                <TableCell>
                  <Chip
                    label={campaign.status}
                    color={getStatusColor(campaign.status)}
                    size="small"
                  />
                </TableCell>
                <TableCell>{campaign.lists?.length || 0}</TableCell>
                <TableCell>{campaign.sent || 0}</TableCell>
                <TableCell>{campaign.views || 0}</TableCell>
                <TableCell>{campaign.clicks || 0}</TableCell>
                <TableCell>
                  {new Date(campaign.createdAt).toLocaleDateString()}
                </TableCell>
                <TableCell align="right">
                  {campaign.status === 'running' ? (
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        updateCampaignStatus(campaign.id, 'paused');
                      }}
                    >
                      <PauseIcon />
                    </IconButton>
                  ) : campaign.status === 'paused' ? (
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        updateCampaignStatus(campaign.id, 'running');
                      }}
                    >
                      <PlayIcon />
                    </IconButton>
                  ) : null}
                  <IconButton
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleMenuOpen(e, campaign);
                    }}
                  >
                    <MoreIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={() => navigate(`/campaigns/${selectedCampaign?.id}`)}>
          Edit
        </MenuItem>
        <MenuItem onClick={() => handleStatusChange('running')}>
          Start
        </MenuItem>
        <MenuItem onClick={() => handleStatusChange('paused')}>
          Pause
        </MenuItem>
        <MenuItem onClick={() => handleStatusChange('cancelled')}>
          Cancel
        </MenuItem>
        <MenuItem onClick={handleDelete}>Delete</MenuItem>
      </Menu>
    </Box>
  );
};
```

### 5.6 Form Handling with React Hook Form

```typescript
// src/pages/Campaigns/CampaignEditor.tsx
import React from 'react';
import { useForm, Controller } from 'react-hook-form';
import { yupResolver } from '@hookform/resolvers/yup';
import * as yup from 'yup';
import {
  Box,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem
} from '@mui/material';
import { useCampaignStore } from '@/stores/campaignStore';
import { useNavigate, useParams } from 'react-router-dom';
import { ListSelector } from '@/components/Forms/ListSelector';
import { RichTextEditor } from '@/components/Editors/RichTextEditor';

const schema = yup.object().shape({
  name: yup.string().required('Name is required'),
  subject: yup.string().required('Subject is required'),
  fromEmail: yup.string().email('Invalid email').required('From email is required'),
  body: yup.string().required('Body is required'),
  listIds: yup.array().min(1, 'Select at least one list').required()
});

export const CampaignEditor: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { selectedCampaign, createCampaign, updateCampaign, fetchCampaign } =
    useCampaignStore();

  const {
    control,
    handleSubmit,
    formState: { errors }
  } = useForm({
    resolver: yupResolver(schema),
    defaultValues: selectedCampaign || {
      name: '',
      subject: '',
      fromEmail: '',
      body: '',
      contentType: 'richtext',
      listIds: []
    }
  });

  React.useEffect(() => {
    if (id) {
      fetchCampaign(parseInt(id));
    }
  }, [id]);

  const onSubmit = async (data: any) => {
    try {
      if (id) {
        await updateCampaign(parseInt(id), data);
      } else {
        await createCampaign(data);
      }
      navigate('/campaigns');
    } catch (error) {
      console.error(error);
    }
  };

  return (
    <Box p={3}>
      <h1>{id ? 'Edit Campaign' : 'New Campaign'}</h1>

      <form onSubmit={handleSubmit(onSubmit)}>
        <Box display="flex" flexDirection="column" gap={3}>
          <Controller
            name="name"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                label="Campaign Name"
                error={!!errors.name}
                helperText={errors.name?.message}
                fullWidth
              />
            )}
          />

          <Controller
            name="subject"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                label="Subject"
                error={!!errors.subject}
                helperText={errors.subject?.message}
                fullWidth
              />
            )}
          />

          <Controller
            name="fromEmail"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                label="From Email"
                type="email"
                error={!!errors.fromEmail}
                helperText={errors.fromEmail?.message}
                fullWidth
              />
            )}
          />

          <Controller
            name="listIds"
            control={control}
            render={({ field }) => (
              <ListSelector
                value={field.value}
                onChange={field.onChange}
                error={errors.listIds?.message}
              />
            )}
          />

          <Controller
            name="contentType"
            control={control}
            render={({ field }) => (
              <FormControl fullWidth>
                <InputLabel>Content Type</InputLabel>
                <Select {...field}>
                  <MenuItem value="richtext">Rich Text</MenuItem>
                  <MenuItem value="html">HTML</MenuItem>
                  <MenuItem value="markdown">Markdown</MenuItem>
                  <MenuItem value="plain">Plain Text</MenuItem>
                </Select>
              </FormControl>
            )}
          />

          <Controller
            name="body"
            control={control}
            render={({ field }) => (
              <RichTextEditor
                value={field.value}
                onChange={field.onChange}
                error={errors.body?.message}
              />
            )}
          />

          <Box display="flex" gap={2}>
            <Button type="submit" variant="contained">
              {id ? 'Update' : 'Create'}
            </Button>
            <Button onClick={() => navigate('/campaigns')}>
              Cancel
            </Button>
          </Box>
        </Box>
      </form>
    </Box>
  );
};
```

---

## 6. ADVANCED FEATURES PORTING

### 6.1 Queue System (Web Job)

**Queue Processor Web Job**:
```python
# webjobs/queue_processor/run.py
import asyncio
import os
from datetime import datetime, timedelta
from typing import List, Dict, Optional

from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
from sqlalchemy import select, update, and_, or_

from app.models.email_queue import EmailQueue
from app.models.campaign import Campaign
from app.manager.campaign_manager import CampaignManager
from app.config import settings

# Create async engine
engine = create_async_engine(settings.DATABASE_URL, echo=False)
async_session = sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

class QueueProcessor:
    def __init__(self, db: AsyncSession):
        self.db = db
        self.batch_size = settings.QUEUE_BATCH_SIZE  # 100
        self.concurrency = settings.CONCURRENCY  # 10
        self.poll_interval = 5  # seconds
        self.manager = CampaignManager(db)

    async def process_batch(self):
        """Process a batch of queued emails."""
        # Check if queue is paused
        if await self._is_queue_paused():
            return

        # Check if within time window
        if not await self._is_within_time_window():
            return

        # Get server capacities
        server_capacities = await self._get_server_capacities()

        # Check account-wide rate limits
        account_limited = await self._check_account_rate_limits()
        if account_limited:
            return

        # Fetch batch of queued emails
        queued_emails = await self._fetch_queued_emails()

        if not queued_emails:
            # No emails to process
            # Check for completed campaigns
            await self._mark_completed_campaigns()
            return

        # Process emails concurrently
        tasks = []
        semaphore = asyncio.Semaphore(self.concurrency)

        for email in queued_emails:
            task = self._process_email(email, server_capacities, semaphore)
            tasks.append(task)

        await asyncio.gather(*tasks, return_exceptions=True)

        # Update campaign stats
        await self._sync_campaign_stats()

    async def _fetch_queued_emails(self) -> List[EmailQueue]:
        """Fetch next batch of queued emails."""
        query = select(EmailQueue).where(
            and_(
                EmailQueue.status == 'queued',
                EmailQueue.scheduled_at <= datetime.utcnow()
            )
        ).order_by(
            EmailQueue.priority.desc(),
            EmailQueue.scheduled_at.asc()
        ).limit(self.batch_size)

        result = await self.db.execute(query)
        return result.scalars().all()

    async def _process_email(
        self,
        email: EmailQueue,
        server_capacities: Dict,
        semaphore: asyncio.Semaphore
    ):
        """Process a single email."""
        async with semaphore:
            try:
                # Atomically claim email
                result = await self.db.execute(
                    update(EmailQueue)
                    .where(
                        and_(
                            EmailQueue.id == email.id,
                            EmailQueue.status == 'queued'
                        )
                    )
                    .values(status='sending')
                )

                if result.rowcount == 0:
                    # Already claimed by another processor
                    return

                await self.db.commit()

                # Select SMTP server with most capacity
                server_uuid = self._select_server(server_capacities)

                if not server_uuid:
                    # No capacity available, requeue
                    await self._requeue_email(email)
                    return

                # Send email via campaign manager
                success = await self.manager.send_email_by_id(
                    email.campaign_id,
                    email.subscriber_id,
                    server_uuid=server_uuid
                )

                if success:
                    # Mark as sent
                    email.status = 'sent'
                    email.sent_at = datetime.utcnow()
                    email.assigned_smtp_server_uuid = server_uuid

                    # Update server usage
                    await self._increment_server_usage(server_uuid)

                    # Update subscriber_last_send for Smart Sending
                    await self._update_subscriber_last_send(email.subscriber_id)

                else:
                    # Mark as failed
                    email.status = 'failed'
                    email.retry_count += 1
                    email.last_error = "Send failed"

                await self.db.commit()

            except Exception as e:
                # Mark as failed
                email.status = 'failed'
                email.retry_count += 1
                email.last_error = str(e)
                await self.db.commit()

    def _select_server(self, capacities: Dict) -> Optional[str]:
        """Select SMTP server with most remaining capacity."""
        best_server = None
        max_remaining = 0

        for server_uuid, capacity in capacities.items():
            daily_remaining = capacity['daily_remaining']
            window_remaining = capacity['window_remaining']

            # Skip if at capacity
            if daily_remaining <= 0 or window_remaining <= 0:
                continue

            # Use minimum of daily and window remaining
            remaining = min(daily_remaining, window_remaining)

            if remaining > max_remaining:
                max_remaining = remaining
                best_server = server_uuid

        return best_server

    async def _get_server_capacities(self) -> Dict:
        """Get current capacity for all SMTP servers."""
        # Query smtp_daily_usage and smtp_rate_limit_state
        # Return dict of {server_uuid: {daily_remaining, window_remaining}}
        pass

    async def _increment_server_usage(self, server_uuid: str):
        """Increment usage counters for SMTP server."""
        # Update smtp_daily_usage
        # Update smtp_rate_limit_state
        pass

    async def _is_queue_paused(self) -> bool:
        """Check if queue processing is paused."""
        # Query settings
        pass

    async def _is_within_time_window(self) -> bool:
        """Check if current time is within send window."""
        # Get send_time_start and send_time_end from settings
        # Compare with current time in configured timezone
        pass

    async def _check_account_rate_limits(self) -> bool:
        """Check account-wide rate limits."""
        # Check account_rate_limit_state table
        # Return True if limited, False if ok
        pass

    async def _mark_completed_campaigns(self):
        """Mark campaigns with no queued/sending emails as finished."""
        # Find campaigns with use_queue=true, status=running
        # Check if any emails still queued/sending
        # If not, mark as finished
        pass

    async def _requeue_email(self, email: EmailQueue):
        """Requeue an email (no capacity)."""
        email.status = 'queued'
        email.scheduled_at = datetime.utcnow() + timedelta(seconds=60)
        await self.db.commit()

    async def _update_subscriber_last_send(self, subscriber_id: int):
        """Update last send timestamp for Smart Sending."""
        # Upsert subscriber_last_send table
        pass

    async def _sync_campaign_stats(self):
        """Sync campaign.sent counters from queue."""
        # Update campaign.sent from email_queue counts
        pass

async def main():
    """Main loop."""
    async with async_session() as db:
        processor = QueueProcessor(db)

        print("Queue Processor started")

        while True:
            try:
                await processor.process_batch()
                await asyncio.sleep(processor.poll_interval)
            except Exception as e:
                print(f"Error in queue processor: {e}")
                await asyncio.sleep(10)

if __name__ == "__main__":
    asyncio.run(main())
```

### 6.2 Azure Event Grid Integration

```python
# app/bounce/webhooks/azure.py
from fastapi import Request, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.azure_delivery_event import AzureDeliveryEvent
from app.models.azure_engagement_event import AzureEngagementEvent
from app.models.bounce import Bounce

async def handle_azure_eventgrid(request: Request, db: AsyncSession):
    """Handle Azure Event Grid webhook."""
    data = await request.json()

    for event in data:
        event_type = event.get('eventType')

        if event_type == 'Microsoft.Communication.EmailDeliveryReportReceived':
            await handle_delivery_event(event, db)
        elif event_type == 'Microsoft.Communication.EmailEngagementTrackingReportReceived':
            await handle_engagement_event(event, db)

async def handle_delivery_event(event: dict, db: AsyncSession):
    """Handle email delivery event."""
    event_data = event['data']

    # Extract message properties
    message_id = event_data.get('messageId')
    status = event_data.get('status')  # Delivered, Bounced, Failed, etc.

    # Extract campaign_id and subscriber_id from custom properties
    custom_props = event_data.get('customProperties', {})
    campaign_id = custom_props.get('campaign_id')
    subscriber_id = custom_props.get('subscriber_id')

    # Record delivery event
    delivery_event = AzureDeliveryEvent(
        azure_message_id=message_id,
        campaign_id=campaign_id,
        subscriber_id=subscriber_id,
        status=status,
        status_reason=event_data.get('statusReason'),
        delivery_status_details=event_data.get('deliveryStatusDetails'),
        event_timestamp=event_data.get('timestamp'),
        created_at=datetime.utcnow()
    )

    db.add(delivery_event)

    # If bounce, record to bounces table
    if status in ['Bounced', 'Failed', 'Suppressed', 'FilteredSpam']:
        bounce_type = 'hard' if status in ['Bounced', 'Suppressed'] else 'soft'

        bounce = Bounce(
            subscriber_id=subscriber_id,
            campaign_id=campaign_id,
            type=bounce_type,
            source='azure',
            meta={'azure_event': event_data},
            created_at=datetime.utcnow()
        )

        db.add(bounce)

        # Check bounce threshold and take action
        await check_bounce_threshold(db, subscriber_id, bounce_type)

    await db.commit()

async def handle_engagement_event(event: dict, db: AsyncSession):
    """Handle email engagement event (open/click)."""
    event_data = event['data']

    message_id = event_data.get('messageId')
    engagement_type = event_data.get('engagementType')  # 'open' or 'click'

    custom_props = event_data.get('customProperties', {})
    campaign_id = custom_props.get('campaign_id')
    subscriber_id = custom_props.get('subscriber_id')

    engagement = AzureEngagementEvent(
        azure_message_id=message_id,
        campaign_id=campaign_id,
        subscriber_id=subscriber_id,
        engagement_type=engagement_type,
        engagement_context=event_data.get('engagementContext'),
        user_agent=event_data.get('userAgent'),
        event_timestamp=event_data.get('timestamp'),
        created_at=datetime.utcnow()
    )

    db.add(engagement)
    await db.commit()
```

### 6.3 Shopify Integration

```python
# app/bounce/webhooks/shopify.py
from fastapi import Request, Header, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
import hmac
import hashlib

from app.models.purchase_attribution import PurchaseAttribution
from app.core.campaigns import find_recent_engagement

async def handle_shopify_order(
    request: Request,
    db: AsyncSession,
    x_shopify_hmac_sha256: str = Header(None)
):
    """Handle Shopify order webhook."""
    # Verify webhook signature
    body = await request.body()
    secret = settings.SHOPIFY_WEBHOOK_SECRET.encode()
    computed_hmac = hmac.new(secret, body, hashlib.sha256).digest().hex()

    if computed_hmac != x_shopify_hmac_sha256:
        raise HTTPException(status_code=401, detail="Invalid signature")

    order_data = await request.json()

    # Extract order info
    order_id = str(order_data['id'])
    customer_email = order_data['customer']['email']
    total_price = float(order_data['total_price'])
    currency = order_data['currency']

    # Find subscriber by email
    subscriber = await get_subscriber_by_email(db, customer_email)

    if not subscriber:
        # No subscriber found, cannot attribute
        return {"status": "no_subscriber"}

    # Look for recent engagement (link click or email open)
    attribution = await find_recent_engagement(
        db,
        subscriber.id,
        attribution_window_days=settings.SHOPIFY_ATTRIBUTION_WINDOW
    )

    if attribution:
        # Create purchase attribution
        purchase = PurchaseAttribution(
            campaign_id=attribution['campaign_id'],
            subscriber_id=subscriber.id,
            order_id=order_id,
            order_number=order_data.get('order_number'),
            customer_email=customer_email,
            total_price=total_price,
            currency=currency,
            attributed_via=attribution['type'],  # 'link_click' or 'email_open'
            confidence=attribution['confidence'],  # 'high' or 'medium'
            shopify_data=order_data,
            created_at=datetime.utcnow()
        )

        db.add(purchase)
        await db.commit()

        return {"status": "attributed", "campaign_id": attribution['campaign_id']}

    return {"status": "no_attribution"}

async def find_recent_engagement(
    db: AsyncSession,
    subscriber_id: int,
    attribution_window_days: int
) -> Optional[Dict]:
    """Find recent campaign engagement for subscriber."""
    cutoff = datetime.utcnow() - timedelta(days=attribution_window_days)

    # Look for link clicks (highest confidence)
    click_query = select(LinkClick).where(
        and_(
            LinkClick.subscriber_id == subscriber_id,
            LinkClick.created_at >= cutoff
        )
    ).order_by(LinkClick.created_at.desc()).limit(1)

    result = await db.execute(click_query)
    click = result.scalar_one_or_none()

    if click:
        return {
            'campaign_id': click.campaign_id,
            'type': 'link_click',
            'confidence': 'high'
        }

    # Look for email opens (medium confidence)
    view_query = select(CampaignView).where(
        and_(
            CampaignView.subscriber_id == subscriber_id,
            CampaignView.created_at >= cutoff
        )
    ).order_by(CampaignView.created_at.desc()).limit(1)

    result = await db.execute(view_query)
    view = result.scalar_one_or_none()

    if view:
        return {
            'campaign_id': view.campaign_id,
            'type': 'email_open',
            'confidence': 'medium'
        }

    return None
```

---

## 7. AZURE DEPLOYMENT ARCHITECTURE

### 7.1 Azure Services Configuration

**Azure App Service (Web App)**:
```yaml
Name: listmonk-app
Location: East US (or preferred region)
Runtime: Python 3.11
OS: Linux
App Service Plan:
  Name: listmonk-plan
  Tier: Premium V3 P1v3 (2 cores, 8GB RAM) - minimum recommended
  Auto-scaling: Enabled
    - Min instances: 2
    - Max instances: 10
    - Scale rules:
      - CPU > 70% for 5 minutes → Scale out
      - CPU < 30% for 10 minutes → Scale in

Configuration:
  WEBSITES_PORT: 8000
  SCM_DO_BUILD_DURING_DEPLOYMENT: true
  WEBSITES_ENABLE_APP_SERVICE_STORAGE: false
```

**Environment Variables** (App Service Configuration):
```bash
# Database
DATABASE_URL=postgresql+asyncpg://listmonkadmin@listmonk-db:PASSWORD@listmonk-db.postgres.database.azure.com:5432/listmonk?ssl=require

# Application
SECRET_KEY=<from Key Vault>
DEBUG=false
ALLOWED_HOSTS=listmonk-app.azurewebsites.net,yourdomain.com
CORS_ORIGINS=https://yourdomain.com

# Azure Services
AZURE_STORAGE_CONNECTION_STRING=<from Key Vault>
AZURE_STORAGE_CONTAINER=media
APPLICATIONINSIGHTS_CONNECTION_STRING=<auto-configured>

# Email Settings
DEFAULT_FROM_EMAIL=noreply@yourdomain.com

# Queue Settings
QUEUE_BATCH_SIZE=100
CONCURRENCY=10
MESSAGE_RATE=500

# SMTP (or reference from database settings table)
# Can be stored in Key Vault or database
```

**Azure Database for PostgreSQL**:
```yaml
Name: listmonk-db
Deployment: Flexible Server
Version: PostgreSQL 14 or 15
Tier: Burstable or General Purpose
  - Burstable B2s (2 vCores, 4GB RAM) - Dev/Test
  - General Purpose D2ds_v4 (2 vCores, 8GB RAM) - Production
Compute: 2-4 vCores
Storage: 128GB-256GB (auto-grow enabled)
Backup:
  Retention: 7-35 days
  Geo-redundant: Optional
High Availability: Optional (zone-redundant)
Connection Security:
  - Allow Azure services: Yes
  - SSL enforcement: Required
  - Min TLS version: 1.2
```

**Azure Blob Storage**:
```yaml
Name: listmonkmedia
Performance: Standard
Replication: LRS (Locally Redundant) or GRS (Geo-Redundant)
Containers:
  - media (public blob access for campaign images)
  - private (private access for attachments)
CORS:
  - Allow origin: https://listmonk-app.azurewebsites.net
  - Allowed methods: GET, POST, PUT, DELETE
  - Allowed headers: *
```

**Azure Key Vault**:
```yaml
Name: listmonk-vault
Secrets:
  - SECRET_KEY
  - DATABASE_PASSWORD
  - AZURE_STORAGE_CONNECTION_STRING
  - SMTP_PASSWORDS (multiple, per server)
  - SENDGRID_API_KEY
  - SHOPIFY_WEBHOOK_SECRET
  - JWT_SECRET_KEY

Access Policies:
  - App Service Managed Identity: Get, List secrets
```

**Azure Application Insights**:
```yaml
Name: listmonk-insights
Type: Application Insights
Connection: Automatic (App Service integration)
Sampling: 100% during pilot, 50% in production
Retention: 90 days
Alerts:
  - HTTP 5xx errors > 5 in 5 minutes
  - Response time > 2 seconds (95th percentile)
  - Failed dependency calls > 10 in 5 minutes
  - Queue processor errors
```

### 7.2 Web Jobs Configuration

**Web Jobs Overview**:
Azure Web Jobs run within the App Service (no additional cost). They share the same compute resources as the web app.

**Job 1: Campaign Processor** (Continuous):
```yaml
Name: campaign-processor
Type: Continuous (always running)
Script: webjobs/campaign_processor/run.py
Singleton: True (only one instance)
Command: python webjobs/campaign_processor/run.py
Logs: App Service logs + Application Insights
Restart: Auto (on failure)
```

**Job 2: Queue Processor** (Continuous):
```yaml
Name: queue-processor
Type: Continuous
Script: webjobs/queue_processor/run.py
Singleton: True
Command: python webjobs/queue_processor/run.py
Poll Interval: 5 seconds
Logs: App Service logs + Application Insights
Restart: Auto
```

**Job 3: Stats Sync** (Triggered - Scheduled):
```yaml
Name: stats-sync
Type: Triggered (scheduled)
Schedule: */5 * * * * (every 5 minutes, CRON format)
Script: webjobs/stats_sync/run.py
Command: python webjobs/stats_sync/run.py
Timeout: 5 minutes
```

**Job 4: Auto-Pause Scheduler** (Triggered - Scheduled):
```yaml
Name: auto-pause
Type: Triggered
Schedule: * * * * * (every 1 minute)
Script: webjobs/auto_pause/run.py
Command: python webjobs/auto_pause/run.py
Timeout: 1 minute
```

**Job 5: Bounce Mailbox Scanner** (Triggered - Scheduled):
```yaml
Name: bounce-scanner
Type: Triggered
Schedule: */15 * * * * (every 15 minutes, configurable)
Script: webjobs/bounce_scanner/run.py
Command: python webjobs/bounce_scanner/run.py
Timeout: 15 minutes
```

**Job 6: Materialized View Refresh** (Triggered - Scheduled):
```yaml
Name: view-refresh
Type: Triggered
Schedule: 0 3 * * * (daily at 3 AM)
Script: webjobs/view_refresh/run.py
Command: python webjobs/view_refresh/run.py
Timeout: 30 minutes
```

### 7.3 Deployment Configuration

**requirements.txt**:
```txt
# Web Framework
fastapi==0.104.1
uvicorn[standard]==0.24.0
python-multipart==0.0.6

# Database
sqlalchemy[asyncio]==2.0.23
asyncpg==0.29.0
alembic==1.12.1
psycopg2-binary==2.9.9

# Validation
pydantic==2.5.0
pydantic-settings==2.1.0
email-validator==2.1.0

# Authentication
python-jose[cryptography]==3.3.0
passlib[bcrypt]==1.7.4
python-multipart==0.0.6

# Email
aiosmtplib==3.0.1
jinja2==3.1.2
markdown==3.5.1
html2text==2020.1.16

# Azure
azure-storage-blob==12.19.0
azure-identity==1.15.0
azure-keyvault-secrets==4.7.0
opencensus-ext-azure==1.1.13

# Scheduling
apscheduler==3.10.4

# HTTP Client
httpx==0.25.2
aiohttp==3.9.1

# Utils
python-dateutil==2.8.2
pytz==2023.3
humps==0.2.2

# Testing (dev)
pytest==7.4.3
pytest-asyncio==0.21.1
pytest-cov==4.1.0
httpx==0.25.2

# Development
black==23.11.0
flake8==6.1.0
mypy==1.7.1
```

**Startup Command** (App Service):
```bash
gunicorn app.main:app --workers 4 --worker-class uvicorn.workers.UvicornWorker --bind 0.0.0.0:8000 --timeout 120
```

**Alternative (for development)**:
```bash
uvicorn app.main:app --host 0.0.0.0 --port 8000 --workers 4
```

### 7.4 CI/CD Pipeline (Azure DevOps or GitHub Actions)

**GitHub Actions Workflow** (.github/workflows/deploy.yml):
```yaml
name: Deploy to Azure

on:
  push:
    branches: [main]
  workflow_dispatch:

env:
  AZURE_WEBAPP_NAME: listmonk-app
  PYTHON_VERSION: '3.11'
  NODE_VERSION: '18'

jobs:
  build-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: ${{ env.PYTHON_VERSION }}

      - name: Install dependencies
        run: |
          python -m pip install --upgrade pip
          pip install -r requirements.txt

      - name: Run tests
        run: |
          pytest tests/

      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: python-app
          path: |
            .
            !frontend/
            !.git/
            !tests/

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ env.NODE_VERSION }}

      - name: Install and build
        working-directory: ./frontend
        run: |
          npm ci
          npm run build

      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: frontend-build
          path: frontend/dist/

  deploy:
    runs-on: ubuntu-latest
    needs: [build-backend, build-frontend]
    steps:
      - name: Download backend artifact
        uses: actions/download-artifact@v3
        with:
          name: python-app
          path: ./app

      - name: Download frontend artifact
        uses: actions/download-artifact@v3
        with:
          name: frontend-build
          path: ./app/frontend/dist

      - name: Azure Login
        uses: azure/login@v1
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}

      - name: Deploy to Azure Web App
        uses: azure/webapps-deploy@v2
        with:
          app-name: ${{ env.AZURE_WEBAPP_NAME }}
          package: ./app

      - name: Run database migrations
        run: |
          az webapp ssh --name ${{ env.AZURE_WEBAPP_NAME }} --resource-group listmonk-rg \
            --command "cd /home/site/wwwroot && alembic upgrade head"
```

### 7.5 Monitoring and Logging

**Application Insights Queries**:

```kusto
// Failed requests in last hour
requests
| where timestamp > ago(1h)
| where success == false
| summarize count() by resultCode, name
| order by count_ desc

// Slow API endpoints
requests
| where timestamp > ago(1h)
| summarize avg(duration), percentile(duration, 95) by name
| where percentile_duration_95 > 2000  // > 2 seconds
| order by percentile_duration_95 desc

// Queue processor errors
traces
| where message contains "queue_processor"
| where severityLevel >= 3  // Warning or above
| summarize count() by message
| order by count_ desc

// Database query performance
dependencies
| where type == "SQL"
| where timestamp > ago(1h)
| summarize avg(duration), percentile(duration, 95) by name
| where percentile_duration_95 > 100  // > 100ms
| order by percentile_duration_95 desc
```

**Custom Metrics** (via Application Insights SDK):
```python
# app/utils/monitoring.py
from opencensus.ext.azure import metrics_exporter
from opencensus.stats import aggregation, measure, stats, view

# Create measures
campaign_send_measure = measure.MeasureInt("campaigns/sent", "Campaigns sent", "emails")
queue_size_measure = measure.MeasureInt("queue/size", "Queue size", "emails")
smtp_usage_measure = measure.MeasureInt("smtp/usage", "SMTP daily usage", "emails")

# Track custom metrics
def track_campaign_send(count: int):
    stats.stats.record([(campaign_send_measure, count)])

def track_queue_size(size: int):
    stats.stats.record([(queue_size_measure, size)])
```

### 7.6 Security Configuration

**App Service Authentication** (optional - for admin UI SSO):
```yaml
Identity Provider: Azure AD / Microsoft
Allow unauthenticated requests: Yes (for API)
Token store: Enabled
Login URL: /.auth/login/aad
```

**Network Security**:
```yaml
# Virtual Network Integration (optional)
VNet: listmonk-vnet
Subnet: app-subnet
Private Endpoint: Optional (for database)

# App Service IP Restrictions (optional)
Allow: Office IP ranges, VPN IP ranges
Deny: All other IPs
```

**SSL/TLS**:
```yaml
Custom Domain: listmonk.yourdomain.com
SSL Certificate: Managed (free) or Custom
Minimum TLS Version: 1.2
HTTPS Only: Enabled
```

**Managed Identity**:
```yaml
System-assigned: Enabled
Permissions:
  - Key Vault: Secret Get, List
  - Storage Account: Blob Data Contributor
  - Database: Contributor (for migrations)
```

### 7.7 Scaling Strategy

**Vertical Scaling** (increase resources):
- Start: P1v3 (2 cores, 8GB RAM)
- Medium: P2v3 (4 cores, 16GB RAM)
- Large: P3v3 (8 cores, 32GB RAM)

**Horizontal Scaling** (add instances):
```yaml
Auto-scale Rules:
  Default: 2 instances (HA)

  Scale Out Rules:
    - CPU > 70% for 5 minutes → Add 1 instance (max 10)
    - Memory > 80% for 5 minutes → Add 1 instance
    - HTTP Queue Length > 100 → Add 1 instance

  Scale In Rules:
    - CPU < 30% for 10 minutes → Remove 1 instance (min 2)
    - Memory < 40% for 10 minutes → Remove 1 instance

Schedule-based:
  - Business Hours (9AM-6PM): Min 3 instances
  - Off Hours: Min 2 instances
```

**Database Scaling**:
```yaml
# Read Replicas (for reporting)
Primary: Read-write (campaigns, subscribers)
Replica: Read-only (analytics queries)

# Connection Pooling
Max Connections: 100 (per App Service instance)
Connection Lifetime: 5 minutes
```

### 7.8 Disaster Recovery

**Backup Strategy**:
```yaml
Database:
  - Automated: Daily, 7-35 day retention
  - Point-in-time restore: Enabled
  - Geo-redundant backup: Optional

Blob Storage:
  - Geo-redundant storage (GRS): Optional
  - Soft delete: 7-14 days

Application Code:
  - Git repository (GitHub/Azure DevOps)
  - Tagged releases
```

**Recovery Procedures**:
```bash
# Database restore to point-in-time
az postgres flexible-server restore \
  --resource-group listmonk-rg \
  --name listmonk-db-restored \
  --source-server listmonk-db \
  --restore-time "2023-11-13T10:30:00Z"

# Blob storage restore
az storage blob restore \
  --account-name listmonkmedia \
  --time-to-restore "2023-11-13T10:30:00Z" \
  --no-wait

# App Service slot swap (rollback)
az webapp deployment slot swap \
  --resource-group listmonk-rg \
  --name listmonk-app \
  --slot staging \
  --target-slot production
```

---

## 8. DEVELOPMENT PHASES

### Phase 1: Foundation (4-6 weeks)

**Goals**:
- Set up development environment
- Create database schema and models
- Implement core API endpoints
- Set up React application scaffold

**Backend Tasks**:
1. **Week 1-2: Project Setup**
   - Initialize FastAPI project structure
   - Set up SQLAlchemy with async support
   - Configure Alembic migrations
   - Create all database models (30+ tables)
   - Set up pytest framework
   - Configure Application Insights

2. **Week 3-4: Core APIs**
   - Implement subscriber CRUD endpoints
   - Implement list CRUD endpoints
   - Implement campaign CRUD endpoints (basic)
   - Implement template CRUD endpoints
   - Set up authentication (JWT)
   - Basic permission checks

3. **Week 5-6: Database & Testing**
   - Run initial migration
   - Seed test data
   - Write unit tests for models
   - Write integration tests for APIs
   - Set up CI/CD pipeline

**Frontend Tasks**:
1. **Week 1-2: Project Setup**
   - Initialize React + TypeScript project with Vite
   - Set up routing (React Router v6)
   - Configure state management (Zustand)
   - Set up Material-UI theme
   - Configure API client (axios)

2. **Week 3-4: Core Components**
   - Login page
   - Main layout (header, sidebar, content)
   - Dashboard page (basic)
   - List management pages
   - Subscriber list page

3. **Week 5-6: Forms & Integration**
   - List form component
   - Subscriber form component
   - Integrate with backend APIs
   - Error handling
   - Loading states

**Deliverables**:
- ✅ Database schema with all tables
- ✅ Core API endpoints (subscribers, lists, campaigns, templates)
- ✅ Basic authentication
- ✅ React app with routing and layout
- ✅ Basic CRUD functionality for lists and subscribers

### Phase 2: Core Features (6-8 weeks)

**Goals**:
- Complete campaign management
- Implement subscriber management with all features
- Build template system
- Create media library
- Implement basic analytics

**Backend Tasks**:
1. **Week 1-2: Campaign Management**
   - Campaign status workflow (draft → running → finished)
   - Campaign-list associations
   - Campaign preview rendering
   - Test email sending
   - Campaign statistics (basic)

2. **Week 3-4: Subscriber Features**
   - Advanced search and filtering
   - Query-based operations
   - Bulk operations (add to lists, blocklist, delete)
   - Export functionality
   - Subscription management (double opt-in)

3. **Week 5-6: Template System**
   - Template CRUD (complete)
   - Jinja2 template rendering
   - Template functions (TrackLink, UnsubscribeURL, etc.)
   - Template preview
   - Default template management

4. **Week 7-8: Media & Analytics**
   - Azure Blob Storage integration
   - Media upload/download
   - Image optimization
   - View tracking (pixel)
   - Click tracking (links table)
   - Basic analytics endpoints

**Frontend Tasks**:
1. **Week 1-2: Campaign UI**
   - Campaign list with filtering
   - Campaign editor form
   - Rich text editor integration (TinyMCE)
   - Campaign preview modal
   - Status change actions

2. **Week 3-4: Subscriber UI**
   - Advanced search interface
   - Query builder
   - Bulk action toolbar
   - Export dialog
   - Subscriber detail view

3. **Week 5-6: Template UI**
   - Template list
   - Template editor
   - Code editor integration (CodeMirror)
   - Template preview
   - Default template selection

4. **Week 7-8: Media & Analytics**
   - Media library grid
   - Upload interface
   - Media selector component
   - Basic analytics dashboard
   - Campaign analytics page

**Deliverables**:
- ✅ Full campaign management (create, edit, preview, test)
- ✅ Complete subscriber management with bulk operations
- ✅ Template system with rendering
- ✅ Media library with Azure Blob Storage
- ✅ Basic view and click tracking

### Phase 3: Advanced Campaign Features (4-6 weeks)

**Goals**:
- Implement multi-SMTP support (30+ servers)
- Build campaign scheduler
- Create advanced editors
- Implement bounce handling
- Add campaign archiving

**Backend Tasks**:
1. **Week 1-2: Multi-SMTP**
   - SMTP server configuration (settings table)
   - Named SMTP servers → dedicated messengers
   - Connection pooling per server
   - SMTP testing endpoint
   - From email per server

2. **Week 3-4: Campaign Scheduler & Sending**
   - Campaign manager implementation
   - Async SMTP sending with aiosmtplib
   - Message rendering with Jinja2
   - Batch fetching of subscribers
   - Concurrent sending with rate limiting
   - Error handling and retry logic

3. **Week 5-6: Bounce Handling**
   - POP3 mailbox scanning (async)
   - Bounce webhook endpoints (SES, SendGrid, Postmark)
   - Bounce type classification
   - Bounce actions (blocklist, delete)
   - Bounce threshold logic

**Frontend Tasks**:
1. **Week 1-2: SMTP Settings**
   - SMTP server list
   - SMTP server form with validation
   - Test SMTP connection button
   - Multiple server management
   - Bounce mailbox linking

2. **Week 3-4: Advanced Editor**
   - Visual email editor integration
   - HTML code editor
   - Markdown editor
   - Editor mode switcher
   - Media insertion

3. **Week 5-6: Bounce UI**
   - Bounce list with filtering
   - Bounce detail view
   - Bounce actions (delete, blocklist)
   - Bounce statistics
   - Bounce settings form

**Deliverables**:
- ✅ Multi-SMTP server support (30+ servers)
- ✅ Campaign sending with async SMTP
- ✅ Advanced campaign editors
- ✅ Bounce handling (POP3 + webhooks)
- ✅ Campaign archiving

### Phase 4: Queue System (6-8 weeks)

**Goals**:
- Implement queue-based email delivery
- Build Azure Web Job for queue processor
- Add capacity management
- Implement time windows
- Add Smart Sending

**Backend Tasks**:
1. **Week 1-2: Queue Infrastructure**
   - Email queue table and model
   - Queue campaign emails (bulk insert)
   - Queue API endpoints (stats, items, control)
   - Cancel/retry queue items

2. **Week 3-4: Queue Processor Web Job**
   - Queue processor implementation
   - Server capacity calculation
   - Daily limit tracking (smtp_daily_usage)
   - Sliding window rate limiting
   - Atomic email claiming (multi-instance safe)

3. **Week 5-6: Advanced Queue Features**
   - Time window enforcement
   - Auto-pause/resume scheduler
   - Campaign stats sync job
   - Smart Sending implementation
   - Account-wide rate limiting

4. **Week 7-8: Testing & Optimization**
   - Load testing with large campaigns
   - Performance optimization
   - Error handling and recovery
   - Monitoring and alerting

**Frontend Tasks**:
1. **Week 1-2: Queue UI**
   - Queue dashboard page
   - Queue item list with filtering
   - Queue statistics display
   - Server capacity monitoring

2. **Week 3-4: Queue Controls**
   - Pause/resume queue button
   - Clear queue action
   - Send all queued emails action
   - Cancel/retry item actions

3. **Week 5-6: Performance Settings**
   - Performance settings page
   - Time window configuration
   - Rate limit settings
   - Smart Sending settings

4. **Week 7-8: Monitoring UI**
   - Real-time queue stats
   - Server capacity visualization
   - Campaign progress tracking
   - Alert notifications

**Deliverables**:
- ✅ Queue-based email delivery system
- ✅ Azure Web Job queue processor
- ✅ Daily limits and rate limiting
- ✅ Time window enforcement
- ✅ Smart Sending
- ✅ Queue management UI

### Phase 5: Advanced Analytics (4-6 weeks)

**Goals**:
- Implement Azure Event Grid integration
- Add Shopify webhook integration
- Build purchase attribution
- Create aggregate performance metrics
- Build comprehensive analytics dashboard

**Backend Tasks**:
1. **Week 1-2: Azure Event Grid**
   - Event Grid webhook endpoint
   - Delivery event handling
   - Engagement event handling
   - azure_delivery_events table
   - azure_engagement_events table

2. **Week 3-4: Shopify Integration**
   - Shopify order webhook endpoint
   - Webhook signature verification
   - Purchase attribution logic
   - purchase_attributions table
   - Find recent engagement queries

3. **Week 5-6: Analytics Endpoints**
   - Campaign analytics endpoints (views, clicks, bounces)
   - Azure delivery stats
   - Purchase stats per campaign
   - Aggregate performance summary
   - Top links by clicks

**Frontend Tasks**:
1. **Week 1-2: Campaign Analytics Page**
   - Time-series charts (views, clicks over time)
   - Metrics cards (totals, rates)
   - Link click breakdown
   - Bounce analysis

2. **Week 3-4: Azure Analytics Component**
   - Delivery status breakdown
   - Engagement metrics
   - Azure-specific insights
   - Comparison with pixel tracking

3. **Week 5-6: Shopify Analytics**
   - Purchase attribution display
   - Revenue per campaign
   - Order rate metrics
   - Attribution confidence levels

4. **Week 5-6: Performance Dashboard**
   - Aggregate metrics across campaigns
   - Date range selector
   - Campaign comparison
   - Export reports

**Deliverables**:
- ✅ Azure Event Grid integration
- ✅ Shopify purchase attribution
- ✅ Comprehensive analytics dashboard
- ✅ Aggregate performance metrics
- ✅ Revenue tracking per campaign

### Phase 6: Admin Features (3-4 weeks)

**Goals**:
- Implement user management
- Build RBAC system
- Add OIDC integration
- Create settings UI
- Build maintenance tools

**Backend Tasks**:
1. **Week 1: User Management**
   - User CRUD endpoints
   - User roles and permissions
   - Password management
   - API token generation

2. **Week 2: RBAC**
   - Role model and queries
   - Permission checking decorators
   - List roles
   - User roles
   - Role hierarchy

3. **Week 3: OIDC & Settings**
   - OIDC authentication flow
   - Settings CRUD endpoints
   - Settings validation
   - Settings hot-reload

4. **Week 4: Maintenance**
   - Materialized view refresh endpoints
   - Delete old analytics
   - Delete orphan subscribers
   - Database maintenance utilities

**Frontend Tasks**:
1. **Week 1: User UI**
   - User list
   - User form (create/edit)
   - User role assignment
   - Password change

2. **Week 2: Roles UI**
   - User roles list
   - List roles list
   - Role form (create/edit)
   - Permission matrix

3. **Week 3: Settings UI**
   - Settings layout with tabs
   - General settings
   - SMTP settings
   - Bounce settings
   - Performance settings
   - Security settings (OIDC)
   - Appearance settings
   - Shopify settings

4. **Week 4: Maintenance & Logs**
   - System logs viewer
   - Webhook logs viewer
   - Maintenance actions
   - Database statistics

**Deliverables**:
- ✅ User management
- ✅ RBAC with user and list roles
- ✅ OIDC integration
- ✅ Complete settings UI
- ✅ Maintenance tools
- ✅ Log viewers

### Phase 7: Import/Export & Public Pages (3-4 weeks)

**Goals**:
- Implement CSV import
- Add data export (GDPR)
- Build public subscription forms
- Create campaign archive
- Implement unsubscribe pages

**Backend Tasks**:
1. **Week 1-2: Import/Export**
   - CSV import with background processing
   - Import status tracking
   - Import logs
   - Data export (subscribers, lists, analytics)
   - GDPR compliance features

2. **Week 3-4: Public Pages**
   - Public subscription form endpoint
   - Subscription management page
   - Unsubscribe endpoint
   - Campaign archive listing
   - Campaign archive view
   - RSS feed
   - Link click redirect with tracking

**Frontend Tasks**:
1. **Week 1-2: Import UI**
   - Import page with file upload
   - Field mapping interface
   - Import progress tracking
   - Import logs display
   - Error handling

2. **Week 3-4: Public Pages (Server-rendered or SPA)**
   - Public subscription form
   - Subscription management page
   - Unsubscribe page
   - Campaign archive listing
   - Campaign archive single view
   - Mobile-responsive design

**Deliverables**:
- ✅ CSV import with background processing
- ✅ Data export functionality
- ✅ Public subscription forms
- ✅ Campaign archive (public)
- ✅ Unsubscribe pages
- ✅ GDPR compliance features

### Phase 8: Testing & Optimization (4-6 weeks)

**Goals**:
- Comprehensive testing (unit, integration, E2E)
- Performance optimization
- Load testing
- Security audit
- Documentation

**Tasks**:
1. **Week 1-2: Testing**
   - Unit tests (backend): 80%+ coverage
   - Unit tests (frontend): 70%+ coverage
   - Integration tests: All critical flows
   - E2E tests (Playwright): Key user journeys
   - API documentation review

2. **Week 3-4: Performance**
   - Database query optimization
   - Index analysis and creation
   - Caching strategy (Redis optional)
   - Frontend bundle optimization
   - Lazy loading
   - Image optimization

3. **Week 5: Load Testing**
   - Campaign send performance (target: 500-1000 emails/sec)
   - Queue processor performance
   - API endpoint stress testing
   - Database connection pool tuning
   - Identify bottlenecks

4. **Week 6: Security Audit**
   - OWASP Top 10 review
   - SQL injection prevention
   - XSS prevention
   - CSRF protection
   - Authentication/authorization review
   - Secrets management review
   - Penetration testing (optional)

**Deliverables**:
- ✅ 80%+ test coverage (backend)
- ✅ 70%+ test coverage (frontend)
- ✅ E2E tests for critical flows
- ✅ Performance benchmarks
- ✅ Load testing results
- ✅ Security audit report

### Phase 9: Deployment & Migration (2-4 weeks)

**Goals**:
- Set up Azure infrastructure
- Deploy application to production
- Migrate data from existing listmonk
- Monitor and fix issues
- User acceptance testing

**Tasks**:
1. **Week 1: Infrastructure Setup**
   - Create Azure resources (App Service, Database, Storage, Key Vault)
   - Configure networking and security
   - Set up CI/CD pipeline
   - Configure monitoring and alerting
   - Test deployment

2. **Week 2: Data Migration**
   - Export data from existing listmonk
   - Transform data if needed
   - Import into new database
   - Verify data integrity
   - Test with production data

3. **Week 3: Production Deployment**
   - Deploy to production
   - Run smoke tests
   - Monitor for errors
   - Performance tuning
   - Fix critical issues

4. **Week 4: Stabilization**
   - User acceptance testing
   - Bug fixes
   - Performance optimization
   - Documentation
   - Training materials

**Deliverables**:
- ✅ Production Azure infrastructure
- ✅ Application deployed to Azure
- ✅ Data migrated from existing listmonk
- ✅ Monitoring and alerting configured
- ✅ User acceptance testing completed
- ✅ Documentation and training materials

### Phase Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| 1. Foundation | 4-6 weeks | Database, Core APIs, React scaffold |
| 2. Core Features | 6-8 weeks | Campaign/Subscriber/Template management, Media, Basic analytics |
| 3. Advanced Campaign | 4-6 weeks | Multi-SMTP, Scheduler, Bounce handling |
| 4. Queue System | 6-8 weeks | Queue processor, Daily limits, Time windows, Smart Sending |
| 5. Advanced Analytics | 4-6 weeks | Azure Event Grid, Shopify, Performance metrics |
| 6. Admin Features | 3-4 weeks | Users, RBAC, OIDC, Settings, Maintenance |
| 7. Import/Export & Public | 3-4 weeks | CSV import, Public forms, Campaign archive |
| 8. Testing & Optimization | 4-6 weeks | Comprehensive testing, Performance, Security |
| 9. Deployment & Migration | 2-4 weeks | Azure setup, Data migration, Production deployment |
| **TOTAL** | **38-56 weeks** | **Full feature parity** |

---

## 9. TECHNOLOGY STACK DETAILS

### 9.1 Backend Dependencies

**Core Framework**:
```txt
fastapi==0.104.1              # Web framework
uvicorn[standard]==0.24.0     # ASGI server
gunicorn==21.2.0              # Production server
python-multipart==0.0.6       # Form data parsing
```

**Database**:
```txt
sqlalchemy[asyncio]==2.0.23   # ORM with async support
asyncpg==0.29.0               # PostgreSQL async driver
psycopg2-binary==2.9.9        # PostgreSQL sync driver (for Alembic)
alembic==1.12.1               # Database migrations
```

**Validation & Settings**:
```txt
pydantic==2.5.0               # Data validation
pydantic-settings==2.1.0      # Settings management
email-validator==2.1.0        # Email validation
python-dotenv==1.0.0          # Environment variables
```

**Authentication & Security**:
```txt
python-jose[cryptography]==3.3.0  # JWT tokens
passlib[bcrypt]==1.7.4        # Password hashing
python-multipart==0.0.6       # OAuth2 form parsing
bcrypt==4.1.1                 # Bcrypt algorithm
cryptography==41.0.7          # Cryptographic functions
```

**Email**:
```txt
aiosmtplib==3.0.1             # Async SMTP client
jinja2==3.1.2                 # Template engine
markdown==3.5.1               # Markdown to HTML
html2text==2020.1.16          # HTML to plain text
email-validator==2.1.0        # Email validation
```

**Azure Services**:
```txt
azure-storage-blob==12.19.0   # Blob Storage
azure-identity==1.15.0        # Managed Identity
azure-keyvault-secrets==4.7.0 # Key Vault
opencensus-ext-azure==1.1.13  # Application Insights
```

**Scheduling & Background Tasks**:
```txt
apscheduler==3.10.4           # Task scheduling
```

**HTTP Clients**:
```txt
httpx==0.25.2                 # Async HTTP client
aiohttp==3.9.1                # Alternative async HTTP client
requests==2.31.0              # Sync HTTP client (for simple cases)
```

**Utilities**:
```txt
python-dateutil==2.8.2        # Date manipulation
pytz==2023.3                  # Timezone support
humps==0.2.2                  # Case conversion (snake_case ↔ camelCase)
pydantic-extra-types==2.1.0   # Extra Pydantic types
```

**Testing**:
```txt
pytest==7.4.3                 # Test framework
pytest-asyncio==0.21.1        # Async test support
pytest-cov==4.1.0             # Coverage reporting
pytest-mock==3.12.0           # Mocking
httpx==0.25.2                 # Test client
faker==20.1.0                 # Fake data generation
```

**Development Tools**:
```txt
black==23.11.0                # Code formatting
flake8==6.1.0                 # Linting
mypy==1.7.1                   # Type checking
isort==5.12.0                 # Import sorting
pre-commit==3.5.0             # Git hooks
```

### 9.2 Frontend Dependencies

**Core Framework**:
```json
{
  "react": "^18.2.0",
  "react-dom": "^18.2.0",
  "typescript": "^5.3.2",
  "@types/react": "^18.2.37",
  "@types/react-dom": "^18.2.15"
}
```

**Build Tool**:
```json
{
  "vite": "^5.0.0",
  "@vitejs/plugin-react": "^4.2.0"
}
```

**Routing**:
```json
{
  "react-router-dom": "^6.20.0",
  "@types/react-router-dom": "^5.3.3"
}
```

**State Management**:
```json
{
  "zustand": "^4.4.7",
  "immer": "^10.0.3"
}
```

**Alternative (Redux)**:
```json
{
  "@reduxjs/toolkit": "^2.0.1",
  "react-redux": "^9.0.2"
}
```

**Data Fetching** (optional, complementary to Zustand):
```json
{
  "@tanstack/react-query": "^5.12.2",
  "@tanstack/react-query-devtools": "^5.12.2"
}
```

**UI Library** (Material-UI):
```json
{
  "@mui/material": "^5.14.18",
  "@mui/icons-material": "^5.14.18",
  "@mui/x-data-grid": "^6.18.4",
  "@mui/x-date-pickers": "^6.18.4",
  "@emotion/react": "^11.11.1",
  "@emotion/styled": "^11.11.0"
}
```

**Alternative (Ant Design)**:
```json
{
  "antd": "^5.11.5",
  "@ant-design/icons": "^5.2.6"
}
```

**HTTP Client**:
```json
{
  "axios": "^1.6.2",
  "humps": "^2.0.1"
}
```

**Forms**:
```json
{
  "react-hook-form": "^7.48.2",
  "@hookform/resolvers": "^3.3.2",
  "yup": "^1.3.3"
}
```

**Editors**:
```json
{
  "@tinymce/tinymce-react": "^4.3.2",
  "@uiw/react-codemirror": "^4.21.21",
  "@codemirror/lang-html": "^6.4.7",
  "@codemirror/lang-javascript": "^6.2.1"
}
```

**Charts**:
```json
{
  "chart.js": "^4.4.0",
  "react-chartjs-2": "^5.2.0"
}
```

**Alternative (Recharts)**:
```json
{
  "recharts": "^2.10.3"
}
```

**Date Handling**:
```json
{
  "date-fns": "^2.30.0",
  "dayjs": "^1.11.10"
}
```

**Utilities**:
```json
{
  "clsx": "^2.0.0",
  "lodash": "^4.17.21",
  "@types/lodash": "^4.14.202"
}
```

**Testing**:
```json
{
  "vitest": "^1.0.1",
  "@testing-library/react": "^14.1.2",
  "@testing-library/jest-dom": "^6.1.5",
  "@testing-library/user-event": "^14.5.1",
  "jsdom": "^23.0.1",
  "@playwright/test": "^1.40.1"
}
```

**Development**:
```json
{
  "eslint": "^8.54.0",
  "eslint-plugin-react": "^7.33.2",
  "eslint-plugin-react-hooks": "^4.6.0",
  "@typescript-eslint/eslint-plugin": "^6.13.1",
  "@typescript-eslint/parser": "^6.13.1",
  "prettier": "^3.1.0"
}
```

### 9.3 Development Tools

**Python Environment**:
```bash
python>=3.11
pip>=23.3
poetry>=1.7.0 (optional, alternative to pip)
```

**Node Environment**:
```bash
node>=18.0.0
npm>=9.0.0
yarn>=1.22.0 (optional)
```

**Database Tools**:
```bash
postgresql-client>=14
pgAdmin 4 (optional GUI)
Azure Data Studio (optional)
```

**Azure CLI**:
```bash
az>=2.54.0
```

**Docker** (for local development):
```bash
docker>=24.0.0
docker-compose>=2.23.0
```

---

## 10. API ENDPOINT MAPPING

### 10.1 Complete Endpoint List

**Campaign Endpoints** (20+ endpoints):

```python
# Campaign CRUD
GET    /api/v1/campaigns                    # List campaigns with pagination
GET    /api/v1/campaigns/{id}               # Get single campaign
POST   /api/v1/campaigns                    # Create campaign
PUT    /api/v1/campaigns/{id}               # Update campaign
DELETE /api/v1/campaigns/{id}               # Delete campaign

# Campaign Actions
PUT    /api/v1/campaigns/{id}/status        # Change status (draft, running, etc.)
POST   /api/v1/campaigns/{id}/test          # Send test emails
PUT    /api/v1/campaigns/{id}/archive       # Update archive settings
POST   /api/v1/campaigns/{id}/content       # Convert content format
POST   /api/v1/campaigns/{id}/remove-sent-today  # Remove sent subscribers

# Campaign Preview
GET    /api/v1/campaigns/{id}/preview       # Preview campaign HTML
POST   /api/v1/campaigns/{id}/preview       # Preview with custom body
GET    /api/v1/campaigns/{id}/archive/preview  # Preview archive

# Campaign Stats
GET    /api/v1/campaigns/running/stats      # Running campaign stats
GET    /api/v1/campaigns/analytics/views    # View analytics
GET    /api/v1/campaigns/analytics/clicks   # Click analytics
GET    /api/v1/campaigns/analytics/bounces  # Bounce analytics
GET    /api/v1/campaigns/analytics/links    # Link click breakdown
GET    /api/v1/campaigns/analytics/azure-delivery  # Azure delivery stats
GET    /api/v1/campaigns/{id}/unsubscribers  # Unsubscribers list
GET    /api/v1/campaigns/{id}/azure-analytics  # Azure analytics
GET    /api/v1/campaigns/{id}/azure-delivery-events  # Azure delivery events
GET    /api/v1/campaigns/{id}/azure-engagement-events  # Azure engagement events
GET    /api/v1/campaigns/{id}/purchases/stats  # Shopify purchase stats
GET    /api/v1/campaigns/performance/summary  # Aggregate performance
```

**Queue Endpoints** (10 endpoints):

```python
GET    /api/v1/queue/items            # List queue items
GET    /api/v1/queue/stats            # Queue statistics
GET    /api/v1/queue/servers          # SMTP server capacity
PUT    /api/v1/queue/{id}/cancel      # Cancel queue item
PUT    /api/v1/queue/{id}/retry       # Retry queue item
POST   /api/v1/queue/clear            # Clear all queued emails
PUT    /api/v1/queue/pause            # Pause/resume queue
POST   /api/v1/queue/send-all         # Send all queued immediately
```

**Subscriber Endpoints** (15+ endpoints):

```python
# Subscriber CRUD
GET    /api/v1/subscribers            # List subscribers
GET    /api/v1/subscribers/{id}       # Get subscriber
POST   /api/v1/subscribers            # Create subscriber
PUT    /api/v1/subscribers/{id}       # Update subscriber
DELETE /api/v1/subscribers/{id}       # Delete subscriber
DELETE /api/v1/subscribers            # Bulk delete

# Subscriber Actions
POST   /api/v1/subscribers/{id}/optin  # Send opt-in confirmation
PUT    /api/v1/subscribers/lists       # Add to lists
PUT    /api/v1/subscribers/query/lists  # Add to lists by query
PUT    /api/v1/subscribers/blocklist   # Blocklist subscribers
PUT    /api/v1/subscribers/query/blocklist  # Blocklist by query
POST   /api/v1/subscribers/query/delete  # Delete by query

# Subscriber Data
GET    /api/v1/subscribers/{id}/bounces  # Get bounces
DELETE /api/v1/subscribers/{id}/bounces  # Delete bounces
GET    /api/v1/subscribers/{id}/azure-delivery-events  # Azure events
GET    /api/v1/subscribers/{id}/azure-engagement-events  # Azure engagement
```

**List Endpoints** (5 endpoints):

```python
GET    /api/v1/lists                  # List all lists
GET    /api/v1/lists/{id}             # Get list
POST   /api/v1/lists                  # Create list
PUT    /api/v1/lists/{id}             # Update list
DELETE /api/v1/lists/{id}             # Delete list
```

**Template Endpoints** (6 endpoints):

```python
GET    /api/v1/templates              # List templates
GET    /api/v1/templates/{id}         # Get template
POST   /api/v1/templates              # Create template
PUT    /api/v1/templates/{id}         # Update template
PUT    /api/v1/templates/{id}/default  # Set as default
DELETE /api/v1/templates/{id}         # Delete template
```

**Media Endpoints** (3 endpoints):

```python
GET    /api/v1/media                  # List media
POST   /api/v1/media                  # Upload media
DELETE /api/v1/media/{id}             # Delete media
```

**Bounce Endpoints** (5 endpoints):

```python
GET    /api/v1/bounces                # List bounces
DELETE /api/v1/bounces/{id}           # Delete bounce
DELETE /api/v1/bounces                # Bulk delete bounces
PUT    /api/v1/bounces/blocklist      # Blocklist bounced subscribers
```

**Webhook Log Endpoints** (4 endpoints):

```python
GET    /api/v1/webhook-logs           # List webhook logs
DELETE /api/v1/webhook-logs           # Delete webhook logs
GET    /api/v1/webhook-logs/export    # Export webhook logs
```

**Webhook Endpoints** (7 endpoints - external):

```python
POST   /webhooks/bounce                # Generic bounce
POST   /webhooks/bounce/ses            # Amazon SES
POST   /webhooks/bounce/sendgrid       # SendGrid
POST   /webhooks/bounce/postmark       # Postmark
POST   /webhooks/bounce/forwardemail   # ForwardEmail
POST   /webhooks/azure/eventgrid       # Azure Event Grid
POST   /webhooks/shopify/orders        # Shopify orders
```

**User & Auth Endpoints** (15 endpoints):

```python
# Authentication
POST   /api/v1/login                  # Login
POST   /api/v1/logout                 # Logout
GET    /api/v1/auth/oidc              # OIDC callback

# User Management
GET    /api/v1/users                  # List users
GET    /api/v1/users/{id}             # Get user
POST   /api/v1/users                  # Create user
PUT    /api/v1/users/{id}             # Update user
DELETE /api/v1/users/{id}             # Delete user

# User Profile
GET    /api/v1/profile                # Get current user profile
PUT    /api/v1/profile                # Update profile
```

**Role Endpoints** (6 endpoints):

```python
GET    /api/v1/roles/users            # List user roles
GET    /api/v1/roles/lists            # List list roles
POST   /api/v1/roles/users            # Create user role
POST   /api/v1/roles/lists            # Create list role
PUT    /api/v1/roles/users/{id}       # Update user role
PUT    /api/v1/roles/lists/{id}       # Update list role
DELETE /api/v1/roles/{id}             # Delete role
```

**Settings Endpoints** (5 endpoints):

```python
GET    /api/v1/config                 # Get server config
GET    /api/v1/settings               # Get settings
PUT    /api/v1/settings               # Update settings
POST   /api/v1/settings/smtp/test     # Test SMTP
GET    /api/v1/logs                   # Get system logs
```

**Import Endpoints** (4 endpoints):

```python
GET    /api/v1/import/subscribers     # Get import status
POST   /api/v1/import/subscribers     # Start import
DELETE /api/v1/import/subscribers     # Stop import
GET    /api/v1/import/subscribers/logs  # Get import logs
```

**Maintenance Endpoints** (4 endpoints):

```python
DELETE /api/v1/maintenance/analytics/{type}  # Delete old analytics
DELETE /api/v1/maintenance/subscribers/{type}  # Delete subscribers
DELETE /api/v1/maintenance/subscriptions/unconfirmed  # Delete unconfirmed
POST   /api/v1/maintenance/views/refresh  # Refresh materialized views
```

**Dashboard Endpoints** (2 endpoints):

```python
GET    /api/v1/dashboard/counts       # Dashboard counts
GET    /api/v1/dashboard/charts       # Dashboard charts
```

**Admin Endpoints** (1 endpoint):

```python
POST   /api/v1/admin/reload           # Reload application
```

**Transactional Email Endpoints** (1 endpoint):

```python
POST   /api/v1/tx                     # Send transactional email
```

**Public Endpoints** (12 endpoints - no auth):

```python
# Health
GET    /api/health                    # Health check

# Subscription Management
GET    /subscription/{campUUID}/{subUUID}  # Subscription page
POST   /subscription/{campUUID}/{subUUID}  # Update preferences
POST   /subscription/optin/{subUUID}  # Confirm opt-in
GET    /subscription/form/{listUUID}  # Public form
POST   /subscription/form             # Submit subscription

# Tracking
GET    /link/{linkUUID}/{campUUID}/{subUUID}  # Track link click
GET    /campaign/{campUUID}/{subUUID}  # View campaign
GET    /campaign/{campUUID}/{subUUID}/px.png  # Track view

# Archive
GET    /archive                       # Campaign archive list
GET    /archive/{slug}                # View archived campaign
GET    /archive.xml                   # RSS feed

# Lists
GET    /public/lists                  # Get public lists
```

**Total: 156+ endpoints**

---

## 11. Component Mapping

This section provides detailed mappings of all Vue 2 components to their React 18 equivalents.

### 11.1 Page Components Mapping

| Vue 2 Component | Location | React 18 Component | Location | Key Changes |
|----------------|----------|-------------------|----------|-------------|
| Dashboard.vue | views/Dashboard.vue | Dashboard.tsx | pages/Dashboard/Dashboard.tsx | Vuex → Zustand, computed → useMemo |
| Lists.vue | views/Lists.vue | ListsPage.tsx | pages/Lists/ListsPage.tsx | b-table → MUI DataGrid |
| ListForm.vue | views/ListForm.vue | ListForm.tsx | pages/Lists/ListForm.tsx | v-model → controlled inputs |
| Subscribers.vue | views/Subscribers.vue | SubscribersPage.tsx | pages/Subscribers/SubscribersPage.tsx | Bulk actions via MUI |
| SubscriberForm.vue | views/SubscriberForm.vue | SubscriberForm.tsx | pages/Subscribers/SubscriberForm.tsx | JSON editor component |
| SubscriberBulkList.vue | views/SubscriberBulkList.vue | BulkManage.tsx | pages/Subscribers/BulkManage.tsx | File upload via react-dropzone |
| Import.vue | views/Import.vue | ImportPage.tsx | pages/Import/ImportPage.tsx | Upload + progress bar |
| Campaigns.vue | views/Campaigns.vue | CampaignsPage.tsx | pages/Campaigns/CampaignsPage.tsx | Status chips, action buttons |
| Campaign.vue | views/Campaign.vue | CampaignForm.tsx | pages/Campaigns/CampaignForm.tsx | TinyMCE integration |
| CampaignAnalytics.vue | views/CampaignAnalytics.vue | Analytics.tsx | pages/Campaigns/Analytics.tsx | Chart.js → recharts |
| Media.vue | views/Media.vue | MediaPage.tsx | pages/Media/MediaPage.tsx | Grid layout, drag-drop upload |
| Templates.vue | views/Templates.vue | TemplatesPage.tsx | pages/Templates/TemplatesPage.tsx | Template preview modal |
| TemplateForm.vue | views/TemplateForm.vue | TemplateForm.tsx | pages/Templates/TemplateForm.tsx | Code editor integration |
| Maintenance.vue | views/Maintenance.vue | MaintenancePage.tsx | pages/Maintenance/MaintenancePage.tsx | Confirmation dialogs |
| Logs.vue | views/Logs.vue | LogsPage.tsx | pages/Logs/LogsPage.tsx | Virtual scroll for logs |
| Settings.vue | views/Settings.vue | SettingsPage.tsx | pages/Settings/SettingsPage.tsx | Tabbed interface |
| Bounces.vue | views/Bounces.vue | BouncesPage.tsx | pages/Bounces/BouncesPage.tsx | Filter controls |
| Users.vue | views/Users.vue | UsersPage.tsx | pages/Users/UsersPage.tsx | Role selector |
| Forms.vue | views/Forms.vue | FormsPage.tsx | pages/Forms/FormsPage.tsx | Form preview |
| FormPage.vue | views/FormPage.vue | PublicForm.tsx | pages/Public/PublicForm.tsx | Standalone page |
| SubscriptionPage.vue | views/SubscriptionPage.vue | SubscriptionManage.tsx | pages/Public/SubscriptionManage.tsx | Preference checkboxes |

### 11.2 Reusable Components Mapping

| Vue 2 Component | React 18 Component | Purpose | Key Differences |
|----------------|-------------------|---------|-----------------|
| Editor.vue | RichTextEditor.tsx | TinyMCE wrapper | useRef for editor instance |
| ListSelector.vue | ListSelector.tsx | Multi-select lists | MUI Autocomplete with checkboxes |
| CampaignPreview.vue | CampaignPreview.tsx | Email preview modal | iframe rendering |
| CampaignStats.vue | CampaignStats.tsx | Stats display | Metric cards |
| CopyText.vue | CopyToClipboard.tsx | Copy button | navigator.clipboard API |
| DateTime.vue | DateTimePicker.tsx | Date/time input | date-fns for formatting |
| EmptyPlaceholder.vue | EmptyState.tsx | Empty state message | Illustration + CTA |
| ListSelectorMini.vue | QuickListSelector.tsx | Compact list picker | Dropdown with search |
| Loading.vue | LoadingSpinner.tsx | Loading indicator | MUI CircularProgress |
| ModalBulkList.vue | BulkListModal.tsx | Bulk list operations | Dialog with table |
| ModalPreview.vue | PreviewModal.tsx | Generic preview | Dialog with iframe |
| Sidebar.vue | Sidebar.tsx | Navigation sidebar | MUI Drawer |
| SubscriberProfile.vue | SubscriberProfile.tsx | Subscriber details | Collapsible sections |

### 11.3 Vue 2 → React 18 Pattern Examples

**Component Structure:**

```vue
<!-- Vue 2: Campaign.vue -->
<template>
  <section>
    <h1>{{ isNew ? 'New Campaign' : campaign.name }}</h1>
    <b-field label="Name">
      <b-input v-model="form.name" required />
    </b-field>
    <b-button @click="handleSave" :loading="loading">Save</b-button>
  </section>
</template>

<script>
export default {
  data() {
    return {
      form: { name: '', subject: '' },
      loading: false,
    };
  },
  computed: {
    isNew() {
      return !this.$route.params.id;
    },
  },
  methods: {
    async handleSave() {
      this.loading = true;
      await this.$api.campaigns.create(this.form);
      this.loading = false;
      this.$router.push('/campaigns');
    },
  },
};
</script>
```

```typescript
// React 18: CampaignForm.tsx
export const CampaignForm: React.FC = () => {
  const { id } = useParams<{ id?: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const { register, handleSubmit } = useForm<CampaignFormData>();

  const isNew = useMemo(() => !id, [id]);

  const onSubmit = async (data: CampaignFormData) => {
    setLoading(true);
    try {
      await campaignApi.create(data);
      navigate('/campaigns');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box component="section">
      <Typography variant="h4">
        {isNew ? 'New Campaign' : 'Edit Campaign'}
      </Typography>
      <form onSubmit={handleSubmit(onSubmit)}>
        <TextField
          label="Name"
          {...register('name', { required: true })}
          fullWidth
        />
        <LoadingButton loading={loading} type="submit">
          Save
        </LoadingButton>
      </form>
    </Box>
  );
};
```

**State Management:**

```javascript
// Vue 2: Vuex store (store/index.js)
export default new Vuex.Store({
  state: {
    campaigns: [],
    loading: false,
  },
  mutations: {
    SET_CAMPAIGNS(state, campaigns) {
      state.campaigns = campaigns;
    },
    SET_LOADING(state, loading) {
      state.loading = loading;
    },
  },
  actions: {
    async fetchCampaigns({ commit }) {
      commit('SET_LOADING', true);
      const campaigns = await api.campaigns.getAll();
      commit('SET_CAMPAIGNS', campaigns);
      commit('SET_LOADING', false);
    },
  },
});
```

```typescript
// React 18: Zustand store (stores/campaignStore.ts)
export const useCampaignStore = create<CampaignState>()(
  devtools((set, get) => ({
    campaigns: [],
    loading: false,

    fetchCampaigns: async (filters?: CampaignFilters) => {
      set({ loading: true, error: null });
      try {
        const campaigns = await campaignApi.getAll(filters);
        set({ campaigns, loading: false });
      } catch (error) {
        set({ error: error.message, loading: false });
      }
    },

    updateCampaignStatus: async (id: number, status: string) => {
      await campaignApi.updateStatus(id, status);
      set((state) => ({
        campaigns: state.campaigns.map((c) =>
          c.id === id ? { ...c, status } : c
        ),
      }));
    },
  }))
);
```

**API Calls:**

```javascript
// Vue 2: API module (api/index.js)
export default {
  campaigns: {
    getAll(params) {
      return http.get('/api/campaigns', { params });
    },
    get(id) {
      return http.get(`/api/campaigns/${id}`);
    },
    create(data) {
      return http.post('/api/campaigns', data);
    },
  },
};
```

```typescript
// React 18: API service (services/campaignApi.ts)
export const campaignApi = {
  getAll: async (filters?: CampaignFilters): Promise<Campaign[]> => {
    const { data } = await axios.get<Campaign[]>('/api/v1/campaigns', {
      params: filters,
    });
    return data;
  },

  get: async (id: number): Promise<Campaign> => {
    const { data } = await axios.get<Campaign>(`/api/v1/campaigns/${id}`);
    return data;
  },

  create: async (campaign: CampaignCreate): Promise<Campaign> => {
    const { data } = await axios.post<Campaign>('/api/v1/campaigns', campaign);
    return data;
  },
};
```

**Routing:**

```javascript
// Vue 2: router/index.js
const routes = [
  {
    path: '/campaigns',
    name: 'campaigns',
    component: () => import('@/views/Campaigns.vue'),
  },
  {
    path: '/campaigns/new',
    name: 'campaign-new',
    component: () => import('@/views/Campaign.vue'),
  },
  {
    path: '/campaigns/:id',
    name: 'campaign',
    component: () => import('@/views/Campaign.vue'),
  },
];
```

```typescript
// React 18: App.tsx
const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { path: 'campaigns', element: <CampaignsPage /> },
      { path: 'campaigns/new', element: <CampaignForm /> },
      { path: 'campaigns/:id', element: <CampaignForm /> },
      { path: 'campaigns/:id/analytics', element: <Analytics /> },
    ],
  },
  {
    path: '/subscription/:campaignUuid/:subscriberUuid',
    element: <SubscriptionManage />,
  },
]);
```

### 11.4 Lifecycle Method Mappings

| Vue 2 | React 18 | Notes |
|-------|---------|-------|
| created() | useEffect(() => {}, []) | Constructor-like logic |
| mounted() | useEffect(() => {}, []) | DOM ready |
| updated() | useEffect(() => {}) | After every render |
| beforeDestroy() | useEffect(() => { return () => {} }) | Cleanup |
| computed | useMemo(() => {}, [deps]) | Derived state |
| watch | useEffect(() => {}, [watchedValue]) | Watch changes |

### 11.5 Event Handling Changes

**Vue 2:**
```vue
<template>
  <b-button @click="handleClick" @keyup.enter="handleEnter">
    Click Me
  </b-button>
</template>

<script>
export default {
  methods: {
    handleClick() {
      this.$emit('action', { type: 'click' });
    },
    handleEnter() {
      this.$emit('action', { type: 'enter' });
    },
  },
};
</script>
```

**React 18:**
```typescript
interface Props {
  onAction: (event: { type: string }) => void;
}

export const MyButton: React.FC<Props> = ({ onAction }) => {
  const handleClick = () => {
    onAction({ type: 'click' });
  };

  const handleKeyUp = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onAction({ type: 'enter' });
    }
  };

  return (
    <Button onClick={handleClick} onKeyUp={handleKeyUp}>
      Click Me
    </Button>
  );
};
```

### 11.6 Form Validation Changes

**Vue 2 (Vuelidate):**
```vue
<template>
  <b-field :type="$v.form.email.$error ? 'is-danger' : ''"
           :message="$v.form.email.$error ? 'Invalid email' : ''">
    <b-input v-model="form.email" @blur="$v.form.email.$touch()" />
  </b-field>
</template>

<script>
import { required, email } from 'vuelidate/lib/validators';

export default {
  data() {
    return { form: { email: '' } };
  },
  validations: {
    form: {
      email: { required, email },
    },
  },
};
</script>
```

**React 18 (React Hook Form + Zod):**
```typescript
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email('Invalid email').min(1, 'Required'),
});

type FormData = z.infer<typeof schema>;

export const MyForm: React.FC = () => {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  return (
    <TextField
      label="Email"
      {...register('email')}
      error={!!errors.email}
      helperText={errors.email?.message}
    />
  );
};
```

---

## 12. Background Jobs Architecture

This section details the implementation of all 6 Azure Web Jobs that replace Go goroutines.

### 12.1 Web Job Overview

| Job Name | Type | Schedule | Purpose | Azure Compute |
|----------|------|----------|---------|---------------|
| campaign-processor | Continuous | N/A (always running) | Process running campaigns | B1 Basic (1 core, 1.75GB) |
| queue-processor | Continuous | Poll every 5s | Send queued emails | B1 Basic (1 core, 1.75GB) |
| stats-sync | Scheduled | */5 * * * * | Sync campaign stats | Consumption (serverless) |
| auto-pause | Scheduled | * * * * * | Auto-pause completed campaigns | Consumption |
| bounce-scanner | Scheduled | */15 * * * * | Scan bounce mailboxes | Consumption |
| view-refresh | Scheduled | 0 3 * * * | Refresh materialized views | Consumption |

### 12.2 Job 1: Campaign Processor (Continuous)

**Purpose**: Process campaigns with messenger='email' (non-queue) in real-time.

**File Structure**:
```
webjobs/campaign_processor/
├── run.py              # Main entry point
├── processor.py        # Core processing logic
├── requirements.txt
└── host.json           # Web Job config
```

**Implementation**:

```python
# webjobs/campaign_processor/run.py
import asyncio
import logging
from datetime import datetime
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
from app.models import Campaign, Subscriber, CampaignView
from app.messenger import get_messenger
from app.core.campaigns import get_campaign_subscribers
import os

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

DATABASE_URL = os.getenv('DATABASE_URL')
engine = create_async_engine(DATABASE_URL, echo=False, pool_size=10)
AsyncSessionLocal = sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

class CampaignProcessor:
    def __init__(self):
        self.running_campaigns = {}  # {campaign_id: task}
        self.messengers = {}  # {messenger_name: messenger_instance}

    async def start(self):
        """Main loop that polls for running campaigns"""
        logger.info("Campaign processor started")

        while True:
            try:
                await self._check_campaigns()
                await asyncio.sleep(5)  # Poll every 5 seconds
            except Exception as e:
                logger.error(f"Error in campaign processor: {e}", exc_info=True)
                await asyncio.sleep(10)

    async def _check_campaigns(self):
        """Check for new running campaigns and stop finished ones"""
        async with AsyncSessionLocal() as db:
            # Get all running campaigns that are NOT queue-based
            result = await db.execute(
                select(Campaign).where(
                    Campaign.status == 'running',
                    Campaign.messenger != 'automatic',
                    Campaign.use_queue == False
                )
            )
            running = {c.id: c for c in result.scalars().all()}

            # Stop campaigns that are no longer running
            for campaign_id in list(self.running_campaigns.keys()):
                if campaign_id not in running:
                    logger.info(f"Stopping campaign {campaign_id}")
                    task = self.running_campaigns.pop(campaign_id)
                    task.cancel()

            # Start new campaigns
            for campaign_id, campaign in running.items():
                if campaign_id not in self.running_campaigns:
                    logger.info(f"Starting campaign {campaign_id}: {campaign.name}")
                    task = asyncio.create_task(
                        self._process_campaign(campaign)
                    )
                    self.running_campaigns[campaign_id] = task

    async def _process_campaign(self, campaign: Campaign):
        """Process a single campaign"""
        try:
            # Get messenger
            messenger = await self._get_messenger(campaign.messenger)

            # Get subscribers
            async with AsyncSessionLocal() as db:
                subscribers = await get_campaign_subscribers(db, campaign.id)
                total = len(subscribers)
                sent = 0

                logger.info(f"Campaign {campaign.id}: Sending to {total} subscribers")

                # Process in batches
                batch_size = 100
                for i in range(0, total, batch_size):
                    batch = subscribers[i:i + batch_size]

                    # Send emails
                    tasks = []
                    for sub in batch:
                        task = self._send_email(messenger, campaign, sub)
                        tasks.append(task)

                    results = await asyncio.gather(*tasks, return_exceptions=True)

                    # Count successes
                    sent += sum(1 for r in results if not isinstance(r, Exception))

                    # Update campaign stats
                    await self._update_stats(db, campaign.id, sent, total)

                    # Rate limiting
                    await asyncio.sleep(0.1)  # 10 emails/second

                # Mark campaign as finished
                await self._finish_campaign(db, campaign.id)
                logger.info(f"Campaign {campaign.id} finished: {sent}/{total} sent")

        except asyncio.CancelledError:
            logger.info(f"Campaign {campaign.id} cancelled")
        except Exception as e:
            logger.error(f"Error processing campaign {campaign.id}: {e}", exc_info=True)

    async def _send_email(self, messenger, campaign: Campaign, subscriber):
        """Send a single email"""
        try:
            # Render email content
            content = await self._render_template(campaign, subscriber)

            # Send via messenger
            await messenger.push({
                'to': subscriber.email,
                'subject': campaign.subject,
                'body': content,
                'campaign_id': campaign.id,
                'subscriber_id': subscriber.id,
            })

            return True
        except Exception as e:
            logger.error(f"Failed to send to {subscriber.email}: {e}")
            return False

    async def _get_messenger(self, messenger_name: str):
        """Get or create messenger instance"""
        if messenger_name not in self.messengers:
            self.messengers[messenger_name] = await get_messenger(messenger_name)
        return self.messengers[messenger_name]

async def main():
    processor = CampaignProcessor()
    await processor.start()

if __name__ == '__main__':
    asyncio.run(main())
```

**Azure Web Job Configuration**:

```json
// webjobs/campaign_processor/host.json
{
  "version": "2.0",
  "logging": {
    "logLevel": {
      "default": "Information"
    }
  },
  "extensions": {
    "http": {
      "routePrefix": ""
    }
  }
}
```

### 12.3 Job 2: Queue Processor (Continuous)

**Purpose**: Process queued emails respecting daily limits, time windows, and capacity.

**File Structure**:
```
webjobs/queue_processor/
├── run.py              # Main entry point
├── processor.py        # Queue processing logic
├── capacity.py         # Capacity calculation
├── requirements.txt
└── host.json
```

**Implementation**:

```python
# webjobs/queue_processor/run.py
import asyncio
import logging
from datetime import datetime, time
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from app.models import EmailQueue, SMTPServer, SMTPDailyUsage, Settings
from app.messenger import get_messenger
import os

logger = logging.getLogger(__name__)

class QueueProcessor:
    def __init__(self):
        self.batch_size = int(os.getenv('QUEUE_BATCH_SIZE', '100'))
        self.concurrency = int(os.getenv('CONCURRENCY', '10'))
        self.poll_interval = int(os.getenv('POLL_INTERVAL', '5'))
        self.messengers = {}
        self.paused = False

    async def start(self):
        """Main processing loop"""
        logger.info("Queue processor started")

        while True:
            try:
                if not self.paused and await self._is_within_time_window():
                    await self._process_batch()
                else:
                    logger.debug("Queue paused or outside time window")

                await asyncio.sleep(self.poll_interval)

            except Exception as e:
                logger.error(f"Error in queue processor: {e}", exc_info=True)
                await asyncio.sleep(30)

    async def _is_within_time_window(self) -> bool:
        """Check if current time is within sending window"""
        async with AsyncSessionLocal() as db:
            settings = await db.get(Settings, 1)

            if not settings.send_time_start or not settings.send_time_end:
                return True  # No time window configured

            now = datetime.now().time()
            start = time.fromisoformat(settings.send_time_start)
            end = time.fromisoformat(settings.send_time_end)

            return start <= now <= end

    async def _process_batch(self):
        """Process a batch of queued emails"""
        async with AsyncSessionLocal() as db:
            # Get server capacities
            capacities = await self._get_server_capacities(db)

            if not capacities:
                logger.warning("No servers have remaining capacity")
                return

            # Fetch queued emails
            result = await db.execute(
                select(EmailQueue)
                .where(
                    EmailQueue.status == 'queued',
                    EmailQueue.scheduled_at <= datetime.utcnow()
                )
                .order_by(EmailQueue.priority.desc(), EmailQueue.scheduled_at)
                .limit(self.batch_size)
            )
            queued_emails = result.scalars().all()

            if not queued_emails:
                return

            logger.info(f"Processing {len(queued_emails)} queued emails")

            # Process concurrently with semaphore
            semaphore = asyncio.Semaphore(self.concurrency)
            tasks = []

            for email in queued_emails:
                task = self._process_email(email, capacities, semaphore)
                tasks.append(task)

            await asyncio.gather(*tasks, return_exceptions=True)

    async def _get_server_capacities(self, db: AsyncSession) -> dict:
        """Get remaining capacity for each SMTP server"""
        result = await db.execute(select(SMTPServer).where(SMTPServer.enabled == True))
        servers = result.scalars().all()

        capacities = {}
        today = datetime.utcnow().date()

        for server in servers:
            if server.daily_limit == 0:
                capacities[server.uuid] = float('inf')  # Unlimited
            else:
                # Get today's usage
                usage_result = await db.execute(
                    select(SMTPDailyUsage).where(
                        SMTPDailyUsage.smtp_server_uuid == server.uuid,
                        SMTPDailyUsage.usage_date == today
                    )
                )
                usage = usage_result.scalar_one_or_none()

                used = usage.emails_sent if usage else 0
                remaining = server.daily_limit - used

                if remaining > 0:
                    capacities[server.uuid] = remaining

        return capacities

    async def _process_email(self, email: EmailQueue, capacities: dict, semaphore):
        """Process a single queued email"""
        async with semaphore:
            async with AsyncSessionLocal() as db:
                try:
                    # Select server with most capacity
                    server_uuid = max(capacities, key=capacities.get)

                    if capacities[server_uuid] <= 0:
                        logger.warning("All servers at capacity")
                        return

                    # Mark as sending
                    email.status = 'sending'
                    email.assigned_smtp_server_uuid = server_uuid
                    await db.commit()

                    # Get messenger
                    messenger_name = f"email-{server_uuid}"
                    messenger = await self._get_messenger(messenger_name)

                    # Send email
                    await messenger.push({
                        'to': email.subscriber_email,
                        'subject': email.subject,
                        'body': email.body,
                        'campaign_id': email.campaign_id,
                        'subscriber_id': email.subscriber_id,
                    })

                    # Mark as sent
                    email.status = 'sent'
                    email.sent_at = datetime.utcnow()

                    # Update usage counter
                    await self._increment_usage(db, server_uuid)

                    # Decrement capacity
                    capacities[server_uuid] -= 1

                    await db.commit()
                    logger.debug(f"Sent email {email.id} via {server_uuid}")

                except Exception as e:
                    logger.error(f"Failed to send email {email.id}: {e}")
                    email.status = 'failed'
                    email.error = str(e)
                    await db.commit()

    async def _increment_usage(self, db: AsyncSession, server_uuid: str):
        """Increment daily usage counter for SMTP server"""
        today = datetime.utcnow().date()

        result = await db.execute(
            select(SMTPDailyUsage).where(
                SMTPDailyUsage.smtp_server_uuid == server_uuid,
                SMTPDailyUsage.usage_date == today
            )
        )
        usage = result.scalar_one_or_none()

        if usage:
            usage.emails_sent += 1
        else:
            usage = SMTPDailyUsage(
                smtp_server_uuid=server_uuid,
                usage_date=today,
                emails_sent=1
            )
            db.add(usage)

async def main():
    processor = QueueProcessor()
    await processor.start()

if __name__ == '__main__':
    asyncio.run(main())
```

### 12.4 Job 3: Stats Sync (Scheduled - Every 5 Minutes)

**Purpose**: Sync campaign statistics from campaign_views and link_clicks to campaigns table.

```python
# webjobs/stats_sync/run.py
import asyncio
import logging
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from app.models import Campaign, CampaignView, LinkClick

logger = logging.getLogger(__name__)

async def sync_campaign_stats():
    """Sync campaign statistics"""
    async with AsyncSessionLocal() as db:
        # Get all campaigns that use queue
        result = await db.execute(
            select(Campaign).where(Campaign.use_queue == True)
        )
        campaigns = result.scalars().all()

        for campaign in campaigns:
            # Count views
            views_result = await db.execute(
                select(func.count(CampaignView.id)).where(
                    CampaignView.campaign_id == campaign.id
                )
            )
            views = views_result.scalar() or 0

            # Count clicks
            clicks_result = await db.execute(
                select(func.count(LinkClick.id)).where(
                    LinkClick.campaign_id == campaign.id
                )
            )
            clicks = clicks_result.scalar() or 0

            # Update campaign
            campaign.views = views
            campaign.clicks = clicks

            logger.debug(f"Campaign {campaign.id}: {views} views, {clicks} clicks")

        await db.commit()
        logger.info(f"Synced stats for {len(campaigns)} campaigns")

async def main():
    logger.info("Stats sync job started")
    await sync_campaign_stats()
    logger.info("Stats sync job completed")

if __name__ == '__main__':
    asyncio.run(main())
```

**Cron Expression**: `*/5 * * * *` (Every 5 minutes)

### 12.5 Job 4: Auto-Pause (Scheduled - Every Minute)

**Purpose**: Automatically pause campaigns when scheduled end time is reached.

```python
# webjobs/auto_pause/run.py
import asyncio
import logging
from datetime import datetime
from sqlalchemy import select
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from app.models import Campaign

logger = logging.getLogger(__name__)

async def auto_pause_campaigns():
    """Auto-pause campaigns that have reached their end time"""
    async with AsyncSessionLocal() as db:
        now = datetime.utcnow()

        # Find running/scheduled campaigns past their end time
        result = await db.execute(
            select(Campaign).where(
                Campaign.status.in_(['running', 'scheduled']),
                Campaign.send_at.isnot(None),
                Campaign.send_at <= now
            )
        )
        campaigns = result.scalars().all()

        for campaign in campaigns:
            logger.info(f"Auto-pausing campaign {campaign.id}: {campaign.name}")
            campaign.status = 'paused'

        await db.commit()

        if campaigns:
            logger.info(f"Auto-paused {len(campaigns)} campaigns")

async def main():
    logger.info("Auto-pause job started")
    await auto_pause_campaigns()
    logger.info("Auto-pause job completed")

if __name__ == '__main__':
    asyncio.run(main())
```

**Cron Expression**: `* * * * *` (Every minute)

### 12.6 Job 5: Bounce Scanner (Scheduled - Every 15 Minutes)

**Purpose**: Scan POP bounce mailboxes and process bounce emails.

```python
# webjobs/bounce_scanner/run.py
import asyncio
import logging
import email
from poplib import POP3, POP3_SSL
from sqlalchemy import select
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from app.models import BounceMailbox, Bounce, Subscriber
from app.bounce.parser import parse_bounce_email

logger = logging.getLogger(__name__)

class BounceScanner:
    async def scan_all_mailboxes(self):
        """Scan all enabled bounce mailboxes"""
        async with AsyncSessionLocal() as db:
            result = await db.execute(
                select(BounceMailbox).where(BounceMailbox.enabled == True)
            )
            mailboxes = result.scalars().all()

            for mailbox in mailboxes:
                try:
                    await self._scan_mailbox(db, mailbox)
                except Exception as e:
                    logger.error(f"Error scanning mailbox {mailbox.name}: {e}")

    async def _scan_mailbox(self, db: AsyncSession, mailbox: BounceMailbox):
        """Scan a single POP mailbox"""
        logger.info(f"Scanning mailbox: {mailbox.name}")

        # Connect to POP server
        if mailbox.tls_enabled:
            pop = POP3_SSL(mailbox.host, mailbox.port)
        else:
            pop = POP3(mailbox.host, mailbox.port)

        try:
            pop.user(mailbox.username)
            pop.pass_(mailbox.password)

            # Get message count
            num_messages = len(pop.list()[1])
            logger.info(f"Found {num_messages} messages in {mailbox.name}")

            # Process each message
            for i in range(1, num_messages + 1):
                try:
                    # Fetch message
                    response, lines, octets = pop.retr(i)
                    msg_data = b'\n'.join(lines)
                    msg = email.message_from_bytes(msg_data)

                    # Parse bounce
                    bounce_info = parse_bounce_email(msg)

                    if bounce_info:
                        # Create bounce record
                        bounce = Bounce(
                            subscriber_id=bounce_info['subscriber_id'],
                            campaign_id=bounce_info.get('campaign_id'),
                            type=bounce_info['type'],  # 'hard' or 'soft'
                            source='mailbox',
                            meta=bounce_info['meta']
                        )
                        db.add(bounce)

                        # Check if subscriber should be blocklisted
                        await self._check_bounce_threshold(
                            db,
                            bounce_info['subscriber_id'],
                            bounce_info['type']
                        )

                        logger.info(f"Recorded {bounce_info['type']} bounce for subscriber {bounce_info['subscriber_id']}")

                    # Delete message from server
                    if mailbox.auth_enabled:
                        pop.dele(i)

                except Exception as e:
                    logger.error(f"Error processing message {i}: {e}")

            await db.commit()

        finally:
            pop.quit()

    async def _check_bounce_threshold(self, db: AsyncSession, subscriber_id: int, bounce_type: str):
        """Check if subscriber should be blocklisted based on bounce threshold"""
        # Count recent bounces
        result = await db.execute(
            select(func.count(Bounce.id)).where(
                Bounce.subscriber_id == subscriber_id,
                Bounce.type == bounce_type,
                Bounce.created_at >= datetime.utcnow() - timedelta(days=30)
            )
        )
        bounce_count = result.scalar()

        # Blocklist if threshold exceeded
        threshold = 5 if bounce_type == 'soft' else 1
        if bounce_count >= threshold:
            subscriber = await db.get(Subscriber, subscriber_id)
            subscriber.status = 'blocklisted'
            logger.warning(f"Blocklisted subscriber {subscriber_id} after {bounce_count} {bounce_type} bounces")

async def main():
    logger.info("Bounce scanner job started")
    scanner = BounceScanner()
    await scanner.scan_all_mailboxes()
    logger.info("Bounce scanner job completed")

if __name__ == '__main__':
    asyncio.run(main())
```

**Cron Expression**: `*/15 * * * *` (Every 15 minutes)

### 12.7 Job 6: View Refresh (Scheduled - Daily at 3 AM)

**Purpose**: Refresh materialized views for analytics.

```python
# webjobs/view_refresh/run.py
import asyncio
import logging
from sqlalchemy import text
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession

logger = logging.getLogger(__name__)

async def refresh_materialized_views():
    """Refresh all materialized views"""
    async with AsyncSessionLocal() as db:
        views = [
            'subscriber_stats',
            'campaign_stats',
            'link_stats',
        ]

        for view in views:
            logger.info(f"Refreshing materialized view: {view}")
            await db.execute(text(f"REFRESH MATERIALIZED VIEW CONCURRENTLY {view}"))

        await db.commit()
        logger.info(f"Refreshed {len(views)} materialized views")

async def main():
    logger.info("View refresh job started")
    await refresh_materialized_views()
    logger.info("View refresh job completed")

if __name__ == '__main__':
    asyncio.run(main())
```

**Cron Expression**: `0 3 * * *` (Daily at 3 AM)

### 12.8 Web Jobs Deployment Configuration

**Azure CLI Deployment**:

```bash
# Create App Service Plan for Web Jobs
az appservice plan create \
  --name listmonk-webjobs-plan \
  --resource-group listmonk-rg \
  --sku B1 \
  --is-linux

# Create Web App for continuous jobs
az webapp create \
  --name listmonk-webjobs \
  --resource-group listmonk-rg \
  --plan listmonk-webjobs-plan \
  --runtime "PYTHON:3.11"

# Deploy continuous jobs
cd webjobs/campaign_processor
zip -r campaign_processor.zip .
az webapp deployment source config-zip \
  --resource-group listmonk-rg \
  --name listmonk-webjobs \
  --src campaign_processor.zip

# Configure scheduled jobs (Function App)
az functionapp create \
  --name listmonk-scheduled-jobs \
  --resource-group listmonk-rg \
  --consumption-plan-location eastus \
  --runtime python \
  --runtime-version 3.11 \
  --storage-account listmonkstorage
```

**Environment Variables** (Applied to all Web Jobs):

```bash
DATABASE_URL=postgresql+asyncpg://user:pass@host/db
AZURE_STORAGE_CONNECTION_STRING=<from Key Vault>
QUEUE_BATCH_SIZE=100
CONCURRENCY=10
POLL_INTERVAL=5
APPINSIGHTS_INSTRUMENTATIONKEY=<from Key Vault>
```

---

## 13. Testing Strategy

This section defines the comprehensive testing approach for the ported application.

### 13.1 Testing Pyramid

```
        /\
       /  \       E2E Tests (10%)
      /____\      - Playwright
     /      \     Integration Tests (30%)
    /        \    - FastAPI TestClient
   /__________\   - Database fixtures
  /            \  Unit Tests (60%)
 /______________\ - Pytest
                  - React Testing Library
```

**Coverage Requirements**:
- Backend: 80% minimum
- Frontend: 70% minimum
- Critical paths: 100% (authentication, email sending, queue processing)

### 13.2 Backend Testing (Python/FastAPI)

**Framework**: pytest + pytest-asyncio + httpx

**File Structure**:
```
tests/
├── conftest.py                 # Fixtures
├── unit/
│   ├── test_campaigns.py
│   ├── test_subscribers.py
│   ├── test_queue.py
│   └── test_messengers.py
├── integration/
│   ├── test_api_campaigns.py
│   ├── test_api_subscribers.py
│   ├── test_queue_processor.py
│   └── test_bounce_handling.py
└── e2e/
    ├── test_campaign_flow.py
    └── test_subscription_flow.py
```

**Example Unit Test**:

```python
# tests/unit/test_campaigns.py
import pytest
from app.core.campaigns import create_campaign, get_campaign
from app.models import Campaign
from app.schemas import CampaignCreate

@pytest.mark.asyncio
async def test_create_campaign(db_session):
    """Test campaign creation"""
    campaign_data = CampaignCreate(
        name="Test Campaign",
        subject="Test Subject",
        body="<p>Test body</p>",
        messenger="email",
        list_ids=[1, 2]
    )

    campaign = await create_campaign(db_session, campaign_data)

    assert campaign.id is not None
    assert campaign.name == "Test Campaign"
    assert campaign.status == "draft"
    assert len(campaign.lists) == 2

@pytest.mark.asyncio
async def test_get_campaign(db_session, sample_campaign):
    """Test fetching a campaign"""
    campaign = await get_campaign(db_session, sample_campaign.id)

    assert campaign is not None
    assert campaign.id == sample_campaign.id
    assert campaign.name == sample_campaign.name

@pytest.mark.asyncio
async def test_get_nonexistent_campaign(db_session):
    """Test fetching non-existent campaign"""
    campaign = await get_campaign(db_session, 99999)
    assert campaign is None
```

**Example Integration Test**:

```python
# tests/integration/test_api_campaigns.py
import pytest
from httpx import AsyncClient
from app.main import app

@pytest.mark.asyncio
async def test_create_campaign_api(async_client: AsyncClient, auth_headers):
    """Test campaign creation via API"""
    response = await async_client.post(
        "/api/v1/campaigns",
        json={
            "name": "API Test Campaign",
            "subject": "Test",
            "body": "<p>Body</p>",
            "messenger": "email",
            "list_ids": [1]
        },
        headers=auth_headers
    )

    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "API Test Campaign"
    assert data["status"] == "draft"

@pytest.mark.asyncio
async def test_update_campaign_status(async_client: AsyncClient, auth_headers, sample_campaign):
    """Test updating campaign status"""
    response = await async_client.put(
        f"/api/v1/campaigns/{sample_campaign.id}/status",
        json={"status": "running"},
        headers=auth_headers
    )

    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "running"

@pytest.mark.asyncio
async def test_unauthorized_access(async_client: AsyncClient):
    """Test that endpoints require authentication"""
    response = await async_client.get("/api/v1/campaigns")
    assert response.status_code == 401
```

**Fixtures** (conftest.py):

```python
# tests/conftest.py
import pytest
import pytest_asyncio
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
from httpx import AsyncClient
from app.main import app
from app.models import Base, Campaign, Subscriber, List
from app.database import get_db

TEST_DATABASE_URL = "postgresql+asyncpg://test:test@localhost/listmonk_test"

@pytest_asyncio.fixture
async def db_engine():
    """Create test database engine"""
    engine = create_async_engine(TEST_DATABASE_URL, echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()

@pytest_asyncio.fixture
async def db_session(db_engine):
    """Create database session for tests"""
    async_session = sessionmaker(
        db_engine, class_=AsyncSession, expire_on_commit=False
    )
    async with async_session() as session:
        yield session

@pytest_asyncio.fixture
async def async_client(db_session):
    """Create async HTTP client"""
    async def override_get_db():
        yield db_session

    app.dependency_overrides[get_db] = override_get_db

    async with AsyncClient(app=app, base_url="http://test") as client:
        yield client

    app.dependency_overrides.clear()

@pytest.fixture
def auth_headers():
    """Generate authentication headers"""
    # In real implementation, generate JWT token
    return {"Authorization": "Bearer test-token"}

@pytest_asyncio.fixture
async def sample_campaign(db_session):
    """Create sample campaign for testing"""
    campaign = Campaign(
        name="Sample Campaign",
        subject="Sample Subject",
        body="<p>Sample body</p>",
        messenger="email",
        status="draft"
    )
    db_session.add(campaign)
    await db_session.commit()
    await db_session.refresh(campaign)
    return campaign

@pytest_asyncio.fixture
async def sample_subscriber(db_session):
    """Create sample subscriber for testing"""
    subscriber = Subscriber(
        email="test@example.com",
        name="Test User",
        status="enabled"
    )
    db_session.add(subscriber)
    await db_session.commit()
    await db_session.refresh(subscriber)
    return subscriber
```

### 13.3 Frontend Testing (React/TypeScript)

**Framework**: Vitest + React Testing Library + MSW (Mock Service Worker)

**File Structure**:
```
frontend/src/
├── __tests__/
│   ├── pages/
│   │   ├── CampaignsPage.test.tsx
│   │   ├── SubscribersPage.test.tsx
│   │   └── Dashboard.test.tsx
│   ├── components/
│   │   ├── CampaignList.test.tsx
│   │   ├── SubscriberForm.test.tsx
│   │   └── ListSelector.test.tsx
│   └── stores/
│       ├── campaignStore.test.ts
│       └── subscriberStore.test.ts
├── __mocks__/
│   ├── handlers.ts             # MSW handlers
│   └── server.ts               # MSW server
└── test-utils.tsx              # Test utilities
```

**Example Component Test**:

```typescript
// src/__tests__/components/CampaignList.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CampaignList } from '@/components/CampaignList';
import { mockCampaigns } from '@/__mocks__/data';

describe('CampaignList', () => {
  it('renders campaign list', async () => {
    render(<CampaignList />);

    await waitFor(() => {
      expect(screen.getByText('Test Campaign 1')).toBeInTheDocument();
      expect(screen.getByText('Test Campaign 2')).toBeInTheDocument();
    });
  });

  it('handles status update', async () => {
    const user = userEvent.setup();
    render(<CampaignList />);

    await waitFor(() => screen.getByText('Test Campaign 1'));

    const startButton = screen.getByRole('button', { name: /start/i });
    await user.click(startButton);

    await waitFor(() => {
      expect(screen.getByText('running')).toBeInTheDocument();
    });
  });

  it('displays loading state', () => {
    render(<CampaignList loading={true} />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('displays empty state when no campaigns', () => {
    render(<CampaignList campaigns={[]} />);
    expect(screen.getByText(/no campaigns/i)).toBeInTheDocument();
  });
});
```

**Example Store Test**:

```typescript
// src/__tests__/stores/campaignStore.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react-dom/test-utils';
import { useCampaignStore } from '@/stores/campaignStore';
import { mockCampaigns } from '@/__mocks__/data';

describe('campaignStore', () => {
  beforeEach(() => {
    // Reset store state
    useCampaignStore.setState({ campaigns: [], loading: false, error: null });
  });

  it('fetches campaigns', async () => {
    const { result } = renderHook(() => useCampaignStore());

    act(() => {
      result.current.fetchCampaigns();
    });

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
      expect(result.current.campaigns).toHaveLength(2);
    });
  });

  it('updates campaign status', async () => {
    const { result } = renderHook(() => useCampaignStore());

    // Pre-populate campaigns
    act(() => {
      useCampaignStore.setState({ campaigns: mockCampaigns });
    });

    await act(async () => {
      await result.current.updateCampaignStatus(1, 'running');
    });

    const updatedCampaign = result.current.campaigns.find(c => c.id === 1);
    expect(updatedCampaign?.status).toBe('running');
  });

  it('handles fetch error', async () => {
    // Mock API error
    server.use(
      rest.get('/api/v1/campaigns', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json({ detail: 'Server error' }));
      })
    );

    const { result } = renderHook(() => useCampaignStore());

    await act(async () => {
      await result.current.fetchCampaigns();
    });

    expect(result.current.error).toBe('Server error');
    expect(result.current.campaigns).toHaveLength(0);
  });
});
```

**MSW Setup**:

```typescript
// src/__mocks__/handlers.ts
import { rest } from 'msw';
import { mockCampaigns, mockSubscribers } from './data';

export const handlers = [
  // Campaigns
  rest.get('/api/v1/campaigns', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(mockCampaigns));
  }),

  rest.post('/api/v1/campaigns', (req, res, ctx) => {
    const newCampaign = { id: 3, ...req.body };
    return res(ctx.status(201), ctx.json(newCampaign));
  }),

  rest.put('/api/v1/campaigns/:id/status', (req, res, ctx) => {
    const { id } = req.params;
    const { status } = req.body as { status: string };
    const campaign = mockCampaigns.find(c => c.id === Number(id));
    return res(ctx.status(200), ctx.json({ ...campaign, status }));
  }),

  // Subscribers
  rest.get('/api/v1/subscribers', (req, res, ctx) => {
    return res(ctx.status(200), ctx.json(mockSubscribers));
  }),
];
```

```typescript
// src/__mocks__/server.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

### 13.4 E2E Testing (Playwright)

**File Structure**:
```
e2e/
├── tests/
│   ├── campaign-lifecycle.spec.ts
│   ├── subscriber-management.spec.ts
│   ├── template-creation.spec.ts
│   └── public-subscription.spec.ts
├── fixtures/
│   ├── auth.ts
│   └── data.ts
└── playwright.config.ts
```

**Example E2E Test**:

```typescript
// e2e/tests/campaign-lifecycle.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Campaign Lifecycle', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('/login');
    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'password');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard');
  });

  test('should create, start, and complete a campaign', async ({ page }) => {
    // Navigate to campaigns
    await page.goto('/campaigns');
    await page.click('text=New Campaign');

    // Fill campaign form
    await page.fill('input[name="name"]', 'E2E Test Campaign');
    await page.fill('input[name="subject"]', 'Test Subject');

    // Use TinyMCE
    const editorFrame = page.frameLocator('iframe.tox-edit-area__iframe');
    await editorFrame.locator('body').fill('Test email body');

    // Select lists
    await page.click('text=Select lists');
    await page.check('input[value="1"]');  // List 1
    await page.click('text=Done');

    // Save as draft
    await page.click('button:has-text("Save")');
    await expect(page.locator('text=Campaign created')).toBeVisible();

    // Start campaign
    await page.click('button:has-text("Start")');
    await page.click('button:has-text("Confirm")');
    await expect(page.locator('text=Campaign started')).toBeVisible();

    // Check status is "running"
    await expect(page.locator('text=running')).toBeVisible();

    // Wait for completion (or mock it)
    await page.waitForTimeout(5000);

    // Verify campaign finished
    await page.reload();
    await expect(page.locator('text=finished')).toBeVisible();

    // Check analytics
    await page.click('text=Analytics');
    await expect(page.locator('text=Sent:')).toBeVisible();
    await expect(page.locator('text=Opens:')).toBeVisible();
  });

  test('should handle queue-based campaign', async ({ page }) => {
    await page.goto('/campaigns/new');

    await page.fill('input[name="name"]', 'Queue Test Campaign');
    await page.fill('input[name="subject"]', 'Queue Test');

    // Select automatic messenger
    await page.selectOption('select[name="messenger"]', 'automatic');

    // Select large list
    await page.click('text=Select lists');
    await page.check('input[value="7"]');  // Large list
    await page.click('text=Done');

    // Save and start
    await page.click('button:has-text("Save")');
    await page.click('button:has-text("Start")');
    await page.click('button:has-text("Confirm")');

    // Check that emails are queued
    await page.goto('/queue');
    await expect(page.locator('text=Queue Test Campaign')).toBeVisible();

    // Check queue stats
    const queuedCount = await page.locator('[data-testid="queued-count"]').textContent();
    expect(Number(queuedCount)).toBeGreaterThan(0);
  });
});
```

**Playwright Configuration**:

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e/tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
```

### 13.5 Performance Testing

**Load Testing with Locust**:

```python
# tests/load/locustfile.py
from locust import HttpUser, task, between

class ListmonkUser(HttpUser):
    wait_time = between(1, 3)

    def on_start(self):
        """Login"""
        response = self.client.post("/api/v1/login", json={
            "email": "test@example.com",
            "password": "password"
        })
        self.token = response.json()["token"]
        self.headers = {"Authorization": f"Bearer {self.token}"}

    @task(3)
    def list_campaigns(self):
        """List campaigns (most common operation)"""
        self.client.get("/api/v1/campaigns", headers=self.headers)

    @task(2)
    def list_subscribers(self):
        """List subscribers"""
        self.client.get("/api/v1/subscribers", headers=self.headers)

    @task(1)
    def create_subscriber(self):
        """Create subscriber"""
        self.client.post("/api/v1/subscribers", headers=self.headers, json={
            "email": f"test{self.user_id}@example.com",
            "name": "Test User",
            "list_ids": [1]
        })

    @task(1)
    def get_dashboard(self):
        """Get dashboard stats"""
        self.client.get("/api/v1/dashboard/counts", headers=self.headers)
```

**Run load test**:
```bash
locust -f tests/load/locustfile.py --host=https://listmonk-app.azurewebsites.net
```

**Performance Benchmarks**:
- API response time: < 200ms (p95)
- Queue processing rate: 100+ emails/second
- Database query time: < 50ms (p95)
- Frontend load time: < 2 seconds (first contentful paint)

### 13.6 Security Testing

**SQL Injection Testing**:
```python
# tests/security/test_sql_injection.py
@pytest.mark.asyncio
async def test_sql_injection_in_search(async_client, auth_headers):
    """Test that SQL injection is prevented"""
    malicious_input = "' OR '1'='1"

    response = await async_client.get(
        f"/api/v1/subscribers?query={malicious_input}",
        headers=auth_headers
    )

    # Should not return all subscribers
    assert response.status_code == 200
    data = response.json()
    assert len(data) == 0  # No results, not all subscribers
```

**XSS Testing**:
```python
@pytest.mark.asyncio
async def test_xss_in_campaign_name(async_client, auth_headers):
    """Test that XSS is sanitized"""
    xss_payload = "<script>alert('xss')</script>"

    response = await async_client.post(
        "/api/v1/campaigns",
        headers=auth_headers,
        json={
            "name": xss_payload,
            "subject": "Test",
            "body": "Test",
            "messenger": "email"
        }
    )

    assert response.status_code == 201
    data = response.json()
    # Should be escaped
    assert "<script>" not in data["name"]
```

### 13.7 Test CI/CD Integration

**GitHub Actions Workflow**:

```yaml
# .github/workflows/test.yml
name: Run Tests

on: [push, pull_request]

jobs:
  backend-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: listmonk_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-python@v4
        with:
          python-version: '3.11'
      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install pytest pytest-asyncio pytest-cov
      - name: Run tests
        run: pytest --cov=app --cov-report=xml
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  frontend-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Install dependencies
        run: cd frontend && npm ci
      - name: Run tests
        run: cd frontend && npm test -- --coverage
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Install Playwright
        run: npx playwright install --with-deps
      - name: Run E2E tests
        run: npx playwright test
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: playwright-report/
```

---

## 14. Migration Strategy

This section outlines the strategy for migrating data from the existing Go/Vue listmonk to the new Python/React application.

### 14.1 Migration Overview

**Migration Approach**: Zero-downtime blue-green deployment

**Phases**:
1. **Data Export** - Export all data from current system
2. **Schema Transformation** - Transform Go structs → SQLAlchemy models
3. **Data Validation** - Verify data integrity
4. **Incremental Import** - Import in batches
5. **Verification** - Compare old vs new
6. **Cutover** - Switch DNS/traffic
7. **Rollback Plan** - Revert if issues arise

**Timeline**: 2-4 weeks for complete migration

### 14.2 Pre-Migration Checklist

- [ ] Backup current production database
- [ ] Set up staging environment with new application
- [ ] Test migration scripts on staging with production copy
- [ ] Document rollback procedures
- [ ] Schedule maintenance window (optional)
- [ ] Notify users of migration timeline
- [ ] Prepare monitoring dashboards
- [ ] Set up parallel running period (both systems active)

### 14.3 Data Export Scripts

**Export All Tables**:

```python
# migration/export_data.py
import asyncio
import asyncpg
import json
from datetime import datetime
from pathlib import Path

class DataExporter:
    def __init__(self, source_db_url: str, export_dir: str):
        self.source_db_url = source_db_url
        self.export_dir = Path(export_dir)
        self.export_dir.mkdir(exist_ok=True)

    async def export_all(self):
        """Export all tables"""
        conn = await asyncpg.connect(self.source_db_url)

        try:
            tables = [
                'users', 'roles', 'user_roles', 'list_roles',
                'subscribers', 'lists', 'subscriber_lists',
                'campaigns', 'campaign_lists', 'campaign_views', 'link_clicks',
                'templates', 'media', 'bounces', 'bounce_mailboxes',
                'settings', 'smtp_servers', 'smtp_daily_usage',
                'email_queue', 'azure_delivery_events',
                'shopify_purchase_attribution', 'webhook_logs'
            ]

            for table in tables:
                await self._export_table(conn, table)

            # Export special: campaigns with queue info
            await self._export_campaigns_with_queue(conn)

            print(f"✓ Exported {len(tables)} tables to {self.export_dir}")

        finally:
            await conn.close()

    async def _export_table(self, conn, table_name: str):
        """Export a single table to JSON"""
        print(f"Exporting {table_name}...")

        rows = await conn.fetch(f"SELECT * FROM {table_name}")

        data = []
        for row in rows:
            # Convert Row to dict, handling special types
            row_dict = dict(row)
            for key, value in row_dict.items():
                if isinstance(value, datetime):
                    row_dict[key] = value.isoformat()
                elif isinstance(value, bytes):
                    row_dict[key] = value.hex()
            data.append(row_dict)

        output_file = self.export_dir / f"{table_name}.json"
        with open(output_file, 'w') as f:
            json.dump(data, f, indent=2, default=str)

        print(f"  → Exported {len(data)} rows")

    async def _export_campaigns_with_queue(self, conn):
        """Export campaigns with queue statistics"""
        query = """
        SELECT
            c.*,
            COUNT(eq.id) FILTER (WHERE eq.status = 'queued') as queued_count,
            COUNT(eq.id) FILTER (WHERE eq.status = 'sent') as sent_via_queue,
            COUNT(eq.id) FILTER (WHERE eq.status = 'failed') as failed_count
        FROM campaigns c
        LEFT JOIN email_queue eq ON c.id = eq.campaign_id
        WHERE c.use_queue = true
        GROUP BY c.id
        """

        rows = await conn.fetch(query)

        data = [dict(row) for row in rows]

        output_file = self.export_dir / "campaigns_queue_stats.json"
        with open(output_file, 'w') as f:
            json.dump(data, f, indent=2, default=str)

async def main():
    exporter = DataExporter(
        source_db_url="postgresql://listmonkadmin:T@intshr3dd3r@listmonk420-db.postgres.database.azure.com/listmonk",
        export_dir="./migration_export"
    )
    await exporter.export_all()

if __name__ == '__main__':
    asyncio.run(main())
```

**Run export**:
```bash
python migration/export_data.py
```

### 14.4 Data Transformation

**Transform and Import**:

```python
# migration/import_data.py
import asyncio
import json
from pathlib import Path
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
from app.models import (
    User, Role, Subscriber, List, Campaign, Template,
    SMTP, BounceMailbox, EmailQueue, AzureDeliveryEvent
)

class DataImporter:
    def __init__(self, target_db_url: str, import_dir: str):
        self.engine = create_async_engine(target_db_url, echo=True)
        self.import_dir = Path(import_dir)
        self.AsyncSessionLocal = sessionmaker(
            self.engine, class_=AsyncSession, expire_on_commit=False
        )

    async def import_all(self):
        """Import all data in correct order (respecting foreign keys)"""
        # Order matters: parent tables before child tables
        import_order = [
            ('users', User),
            ('roles', Role),
            ('subscribers', Subscriber),
            ('lists', List),
            ('subscriber_lists', None),  # Many-to-many handled separately
            ('templates', Template),
            ('smtp_servers', SMTP),
            ('bounce_mailboxes', BounceMailbox),
            ('campaigns', Campaign),
            ('campaign_lists', None),
            ('campaign_views', None),
            ('link_clicks', None),
            ('email_queue', EmailQueue),
            ('azure_delivery_events', AzureDeliveryEvent),
            ('bounces', None),
        ]

        async with self.AsyncSessionLocal() as db:
            for table_name, model_class in import_order:
                await self._import_table(db, table_name, model_class)

            await db.commit()

        print("✓ Import completed successfully")

    async def _import_table(self, db: AsyncSession, table_name: str, model_class):
        """Import a single table"""
        file_path = self.import_dir / f"{table_name}.json"

        if not file_path.exists():
            print(f"Skipping {table_name} (file not found)")
            return

        print(f"Importing {table_name}...")

        with open(file_path) as f:
            data = json.load(f)

        if model_class:
            # Use SQLAlchemy model
            for row in data:
                # Transform field names if needed
                transformed = self._transform_row(row, model_class)
                instance = model_class(**transformed)
                db.add(instance)
        else:
            # Raw SQL for many-to-many tables
            await self._import_raw_sql(db, table_name, data)

        await db.flush()
        print(f"  → Imported {len(data)} rows")

    def _transform_row(self, row: dict, model_class) -> dict:
        """Transform row to match new schema"""
        transformed = {}

        for key, value in row.items():
            # Handle renamed fields
            if key == 'uuid' and hasattr(model_class, 'uuid'):
                transformed['uuid'] = value
            elif key == 'created_at':
                from datetime import datetime
                transformed['created_at'] = datetime.fromisoformat(value)
            elif key == 'updated_at':
                from datetime import datetime
                transformed['updated_at'] = datetime.fromisoformat(value)
            else:
                transformed[key] = value

        return transformed

    async def _import_raw_sql(self, db: AsyncSession, table_name: str, data: list):
        """Import using raw SQL for performance"""
        if not data:
            return

        # Build INSERT statement
        columns = list(data[0].keys())
        placeholders = ', '.join([f":{col}" for col in columns])
        column_names = ', '.join(columns)

        sql = f"INSERT INTO {table_name} ({column_names}) VALUES ({placeholders})"

        for row in data:
            await db.execute(text(sql), row)

async def main():
    importer = DataImporter(
        target_db_url="postgresql+asyncpg://user:pass@new-host/listmonk",
        import_dir="./migration_export"
    )
    await importer.import_all()

if __name__ == '__main__':
    asyncio.run(main())
```

### 14.5 Data Validation

**Verify Migration**:

```python
# migration/validate_migration.py
import asyncio
import asyncpg
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy import select, func
from app.models import Campaign, Subscriber, List

class MigrationValidator:
    def __init__(self, source_db_url: str, target_db_url: str):
        self.source_conn = None
        self.target_engine = create_async_engine(target_db_url)
        self.source_db_url = source_db_url

    async def validate_all(self):
        """Run all validation checks"""
        self.source_conn = await asyncpg.connect(self.source_db_url)

        try:
            checks = [
                self._validate_counts,
                self._validate_subscribers,
                self._validate_campaigns,
                self._validate_relationships,
                self._validate_queue_data,
            ]

            all_passed = True
            for check in checks:
                passed = await check()
                if not passed:
                    all_passed = False

            if all_passed:
                print("\n✓ All validation checks passed!")
            else:
                print("\n✗ Some validation checks failed")
                return False

            return True

        finally:
            await self.source_conn.close()
            await self.target_engine.dispose()

    async def _validate_counts(self) -> bool:
        """Validate table row counts match"""
        print("\n=== Validating Counts ===")

        tables = ['subscribers', 'lists', 'campaigns', 'templates', 'users']

        all_match = True
        for table in tables:
            source_count = await self.source_conn.fetchval(
                f"SELECT COUNT(*) FROM {table}"
            )

            async with AsyncSession(self.target_engine) as db:
                result = await db.execute(
                    text(f"SELECT COUNT(*) FROM {table}")
                )
                target_count = result.scalar()

            match = source_count == target_count
            status = "✓" if match else "✗"
            print(f"{status} {table}: {source_count} → {target_count}")

            if not match:
                all_match = False

        return all_match

    async def _validate_subscribers(self) -> bool:
        """Validate subscriber data integrity"""
        print("\n=== Validating Subscribers ===")

        # Check random sample
        source_sample = await self.source_conn.fetch(
            "SELECT id, email, name, status FROM subscribers ORDER BY RANDOM() LIMIT 100"
        )

        all_match = True
        async with AsyncSession(self.target_engine) as db:
            for row in source_sample:
                result = await db.execute(
                    select(Subscriber).where(Subscriber.id == row['id'])
                )
                target_sub = result.scalar_one_or_none()

                if not target_sub:
                    print(f"✗ Missing subscriber: {row['email']}")
                    all_match = False
                    continue

                if target_sub.email != row['email'] or target_sub.status != row['status']:
                    print(f"✗ Mismatch for subscriber {row['id']}: {row['email']}")
                    all_match = False

        if all_match:
            print("✓ Subscriber data matches")

        return all_match

    async def _validate_campaigns(self) -> bool:
        """Validate campaign data"""
        print("\n=== Validating Campaigns ===")

        source_campaigns = await self.source_conn.fetch(
            "SELECT id, name, status, sent, use_queue FROM campaigns"
        )

        all_match = True
        async with AsyncSession(self.target_engine) as db:
            for row in source_campaigns:
                result = await db.execute(
                    select(Campaign).where(Campaign.id == row['id'])
                )
                target_camp = result.scalar_one_or_none()

                if not target_camp:
                    print(f"✗ Missing campaign: {row['name']}")
                    all_match = False
                    continue

                if target_camp.sent != row['sent']:
                    print(f"✗ Sent count mismatch for campaign {row['id']}: {row['sent']} → {target_camp.sent}")
                    all_match = False

        if all_match:
            print("✓ Campaign data matches")

        return all_match

    async def _validate_relationships(self) -> bool:
        """Validate many-to-many relationships"""
        print("\n=== Validating Relationships ===")

        # Subscriber-List relationships
        source_subs_lists = await self.source_conn.fetchval(
            "SELECT COUNT(*) FROM subscriber_lists"
        )

        async with AsyncSession(self.target_engine) as db:
            result = await db.execute(
                text("SELECT COUNT(*) FROM subscriber_lists")
            )
            target_subs_lists = result.scalar()

        match = source_subs_lists == target_subs_lists
        status = "✓" if match else "✗"
        print(f"{status} subscriber_lists: {source_subs_lists} → {target_subs_lists}")

        return match

    async def _validate_queue_data(self) -> bool:
        """Validate queue-based campaign data"""
        print("\n=== Validating Queue Data ===")

        source_queue = await self.source_conn.fetchval(
            "SELECT COUNT(*) FROM email_queue"
        )

        async with AsyncSession(self.target_engine) as db:
            result = await db.execute(
                select(func.count(EmailQueue.id))
            )
            target_queue = result.scalar()

        match = source_queue == target_queue
        status = "✓" if match else "✗"
        print(f"{status} email_queue: {source_queue} → {target_queue}")

        return match

async def main():
    validator = MigrationValidator(
        source_db_url="postgresql://listmonkadmin:T@intshr3dd3r@listmonk420-db.postgres.database.azure.com/listmonk",
        target_db_url="postgresql+asyncpg://user:pass@new-host/listmonk"
    )
    success = await validator.validate_all()
    exit(0 if success else 1)

if __name__ == '__main__':
    asyncio.run(main())
```

### 14.6 Cutover Plan

**Blue-Green Deployment Steps**:

1. **Prepare New Environment** (Green):
   ```bash
   # Deploy new Python/React app to Azure App Service
   az webapp deploy --resource-group listmonk-rg --name listmonk-new-app

   # Run migrations
   alembic upgrade head

   # Import data
   python migration/import_data.py

   # Validate
   python migration/validate_migration.py
   ```

2. **Parallel Running Period** (Both systems active):
   - Old system (Blue): `list.bobbyseamoss.com`
   - New system (Green): `list-new.bobbyseamoss.com`
   - Run both for 1 week, comparing metrics
   - Incremental data sync (subscribers, campaigns created during this period)

3. **Incremental Sync Script**:
   ```python
   # migration/incremental_sync.py
   async def sync_new_data(last_sync_time):
       """Sync data created after migration"""
       # Sync new subscribers
       new_subscribers = await source_db.fetch(
           "SELECT * FROM subscribers WHERE created_at > $1",
           last_sync_time
       )
       # ... import to target

       # Sync new campaigns
       new_campaigns = await source_db.fetch(
           "SELECT * FROM campaigns WHERE created_at > $1",
           last_sync_time
       )
       # ... import to target
   ```

4. **Traffic Cutover**:
   ```bash
   # Update DNS to point to new system
   # list.bobbyseamoss.com → listmonk-new-app.azurewebsites.net

   # Or use Azure Traffic Manager for gradual rollout
   az network traffic-manager endpoint update \
     --name listmonk-new \
     --profile-name listmonk-traffic \
     --resource-group listmonk-rg \
     --type azureEndpoints \
     --weight 100
   ```

5. **Monitor New System**:
   - Check Application Insights for errors
   - Verify campaign sends are working
   - Check queue processor is running
   - Monitor database performance

6. **Decommission Old System** (After 2 weeks):
   ```bash
   # Stop old Docker containers
   docker-compose down

   # Archive old database
   pg_dump listmonk > listmonk_old_backup.sql
   ```

### 14.7 Rollback Procedure

**If issues are detected after cutover**:

1. **Immediate Rollback** (< 1 hour):
   ```bash
   # Revert DNS
   # list.bobbyseamoss.com → old-server

   # Or update Traffic Manager
   az network traffic-manager endpoint update \
     --name listmonk-old \
     --weight 100
   ```

2. **Data Sync Back** (If new data was created):
   ```python
   # Export data from new system created after cutover
   async def export_post_cutover_data(cutover_time):
       # Export subscribers created in new system
       new_subs = await new_db.fetch(
           "SELECT * FROM subscribers WHERE created_at > $1",
           cutover_time
       )
       # Import back to old system
   ```

3. **Root Cause Analysis**:
   - Check Application Insights logs
   - Review database query performance
   - Check Web Jobs are running
   - Verify SMTP connectivity

### 14.8 Post-Migration Tasks

- [ ] Update documentation with new URLs
- [ ] Archive old system database
- [ ] Update monitoring dashboards
- [ ] Send user communication about migration
- [ ] Review performance metrics and optimize
- [ ] Clean up temporary migration scripts
- [ ] Update CI/CD pipelines
- [ ] Celebrate successful migration! 🎉

---

## 15. Risks and Mitigation

This section identifies potential risks and mitigation strategies for the porting project.

### 15.1 Technical Risks

#### Risk 1: Performance Degradation

**Description**: Python async performance may not match Go's goroutines

**Likelihood**: Medium | **Impact**: High

**Mitigation**:
- Use async/await throughout the stack
- Implement connection pooling (SQLAlchemy pool_size=25)
- Use Redis for caching frequently accessed data
- Profile code with `py-spy` and `cProfile`
- Load test with Locust before production
- Optimize database queries (proper indexes, EXPLAIN ANALYZE)

**Contingency**:
- Scale horizontally (increase Azure App Service instances)
- Upgrade to P2v3 App Service Plan (4 cores, 16GB RAM)
- Implement CDN for static assets
- Use Azure Redis Cache for session storage

#### Risk 2: Email Delivery Issues

**Description**: SMTP connections, queue processing, or rate limiting may fail

**Likelihood**: Medium | **Impact**: Critical

**Mitigation**:
- Comprehensive testing of all messengers before launch
- Implement circuit breaker pattern for SMTP failures
- Add detailed logging for email send attempts
- Test with multiple SMTP providers (SendGrid, Mailgun, AWS SES)
- Implement retry logic with exponential backoff
- Monitor bounce rates and delivery metrics

**Contingency**:
- Keep old system running in parallel initially
- Implement emergency fallback to direct SMTP send (bypass queue)
- Add manual intervention queue for failed emails

#### Risk 3: Data Migration Failures

**Description**: Data loss or corruption during migration from Go → Python

**Likelihood**: Low | **Impact**: Critical

**Mitigation**:
- Multiple database backups before migration
- Dry-run migrations on staging environment
- Automated validation scripts (validate_migration.py)
- Incremental migration approach (import in batches)
- Checksum verification for critical tables
- Test rollback procedures before cutover

**Contingency**:
- Immediate rollback to old system if issues detected
- Data reconciliation scripts to sync missing records
- Manual data correction for edge cases

#### Risk 4: Azure Service Limitations

**Description**: Web Jobs may not provide same reliability as Go goroutines

**Likelihood**: Medium | **Impact**: Medium

**Mitigation**:
- Use Azure Web Jobs with Always On enabled
- Implement health check endpoints for all jobs
- Set up Azure Monitor alerts for job failures
- Add job restart logic (retry on failure)
- Log all job executions to Application Insights
- Test job reliability under load

**Contingency**:
- Move critical jobs to Azure Functions (serverless)
- Implement external monitoring (UptimeRobot)
- Add on-call rotation for job failures

### 15.2 Timeline Risks

#### Risk 5: Scope Creep

**Description**: Feature additions during development extend timeline

**Likelihood**: High | **Impact**: Medium

**Mitigation**:
- Strict feature freeze after requirements phase
- Change request process for new features
- Separate backlog for post-launch features
- Weekly stakeholder check-ins to align priorities
- Clearly define MVP vs. nice-to-have features

**Contingency**:
- Defer non-critical features to Phase 2
- Add buffer time to schedule (10-20%)
- Hire additional developers if timeline is critical

#### Risk 6: Dependency on External Libraries

**Description**: FastAPI, SQLAlchemy, or React libraries may have breaking changes

**Likelihood**: Low | **Impact**: Medium

**Mitigation**:
- Pin exact versions in requirements.txt and package.json
- Test upgrades in staging before production
- Subscribe to security advisories for dependencies
- Use Dependabot for automated dependency updates
- Maintain test suite to catch breaking changes

**Contingency**:
- Fork critical libraries if necessary
- Vendor dependencies for stability
- Budget time for dependency upgrades

### 15.3 Resource Risks

#### Risk 7: Developer Availability

**Description**: Key developers leave or are unavailable during project

**Likelihood**: Medium | **Impact**: High

**Mitigation**:
- Cross-train team members on all components
- Comprehensive documentation (this porting plan!)
- Code reviews to spread knowledge
- Onboarding documentation for new developers
- Use clear coding standards and patterns

**Contingency**:
- Hire contractors for specific skills (React, Python, Azure)
- Extend timeline if team size reduced
- Prioritize critical path features

#### Risk 8: Azure Cost Overruns

**Description**: Azure services cost more than budgeted ($270-950/month)

**Likelihood**: Medium | **Impact**: Low

**Mitigation**:
- Set up Azure Cost Management alerts
- Right-size App Service Plans based on load testing
- Use Reserved Instances for 1-year commitment (save 30-40%)
- Monitor resource utilization weekly
- Optimize database queries to reduce compute

**Contingency**:
- Scale down to lower App Service tier (B2 instead of P1v3)
- Use Azure Hybrid Benefit if Windows Server licenses available
- Implement aggressive caching to reduce database load

### 15.4 Security Risks

#### Risk 9: SQL Injection or XSS Vulnerabilities

**Description**: New codebase may introduce security vulnerabilities

**Likelihood**: Medium | **Impact**: Critical

**Mitigation**:
- Use parameterized queries (SQLAlchemy prevents SQL injection)
- Sanitize all user inputs (Pydantic validation)
- Use Content Security Policy headers
- Regular security audits (OWASP ZAP, Bandit)
- Penetration testing before launch
- Security code reviews for all pull requests

**Contingency**:
- Immediate hotfix process for security issues
- Bug bounty program for vulnerability discovery
- Web Application Firewall (Azure Application Gateway)

#### Risk 10: Authentication/Authorization Bypass

**Description**: RBAC or OIDC integration may have flaws

**Likelihood**: Low | **Impact**: Critical

**Mitigation**:
- Comprehensive testing of all permission checks
- Use battle-tested libraries (python-jose, passlib)
- Code review all authentication/authorization code
- Test with multiple user roles
- Implement rate limiting on auth endpoints
- Enable MFA for admin users

**Contingency**:
- Disable affected features until fix deployed
- Force password reset for all users if compromise detected
- Audit logs for unauthorized access

### 15.5 Business Risks

#### Risk 11: User Adoption Resistance

**Description**: Users may resist new UI or workflow changes

**Likelihood**: Medium | **Impact**: Medium

**Mitigation**:
- Keep UI similar to existing system (Vue → React)
- Provide migration guide and training
- Beta testing period with power users
- Collect feedback and iterate
- Highlight improvements (performance, features)

**Contingency**:
- Keep old system available for 30 days
- Add "classic view" option if possible
- Provide 1-on-1 training for key users

#### Risk 12: Email Deliverability Issues

**Description**: New system may be flagged as spam or have lower deliverability

**Likelihood**: Low | **Impact**: High

**Mitigation**:
- Warm up new IP addresses gradually
- Maintain SPF/DKIM/DMARC records
- Monitor bounce and spam complaint rates
- Use same SMTP servers as current system
- Test with mail-tester.com before launch

**Contingency**:
- Switch back to old system's SMTP servers
- Work with Azure support for IP reputation
- Implement dedicated IP addresses

### 15.6 Risk Matrix

| Risk | Likelihood | Impact | Priority | Mitigation Status |
|------|------------|--------|----------|-------------------|
| Performance Degradation | Medium | High | **High** | Planned |
| Email Delivery Issues | Medium | Critical | **Critical** | Planned |
| Data Migration Failures | Low | Critical | **Critical** | Planned |
| Azure Service Limitations | Medium | Medium | Medium | Planned |
| Scope Creep | High | Medium | **High** | Planned |
| Dependency Issues | Low | Medium | Low | Planned |
| Developer Availability | Medium | High | **High** | Planned |
| Azure Cost Overruns | Medium | Low | Low | Planned |
| Security Vulnerabilities | Medium | Critical | **Critical** | Planned |
| Auth Bypass | Low | Critical | **Critical** | Planned |
| User Adoption Resistance | Medium | Medium | Medium | Planned |
| Email Deliverability | Low | High | **High** | Planned |

**Priority Levels**:
- **Critical**: Requires immediate mitigation plan and dedicated resources
- **High**: Requires detailed mitigation plan and monitoring
- Medium: Standard mitigation practices sufficient
- Low: Acknowledge and monitor

### 15.7 Risk Review Process

**Weekly Risk Review**:
- Review risk register with project team
- Update likelihood/impact based on current status
- Add newly identified risks
- Track mitigation progress
- Escalate critical risks to stakeholders

**Risk Owners**:
- **Technical Lead**: Performance, email delivery, security
- **Project Manager**: Timeline, scope, resources
- **DevOps Engineer**: Azure services, deployment
- **Product Owner**: User adoption, business risks

---

## 16. Conclusion

This comprehensive porting plan provides a complete blueprint for migrating the listmonk application from Go/Vue 2/Docker to Python/React/Azure App Service with full feature parity.

### 16.1 Executive Summary

**Project Scope**:
- Port **10,130 lines** of Go backend code to Python/FastAPI
- Port **15,000-20,000 lines** of Vue 2 frontend code to React 18
- Migrate **30+ database tables** to SQLAlchemy models
- Implement **156+ API endpoints** in FastAPI
- Deploy **6 background jobs** as Azure Web Jobs
- Maintain **100% feature parity** with existing system

**Timeline**: 38-56 weeks (9-14 months)

**Budget Estimate**:
- Azure Services: $270-950/month
- Development Team: 5-6 developers
- Total Project Cost: ~$400K-600K (assuming $100K/developer/year)

**Key Deliverables**:
1. Fully functional Python/React application
2. Zero-downtime migration from existing system
3. Comprehensive test suite (80%+ coverage)
4. Complete documentation
5. Azure deployment automation (CI/CD)
6. Monitoring and alerting setup

### 16.2 Technology Stack Summary

**Backend**:
- **Framework**: FastAPI 0.104+ (async/await)
- **ORM**: SQLAlchemy 2.0+ with asyncpg
- **Database**: PostgreSQL 15 on Azure Flexible Server
- **Background Jobs**: Azure Web Jobs (continuous + scheduled)
- **Authentication**: JWT + session-based, OIDC support
- **Email**: aiosmtplib for async SMTP
- **Validation**: Pydantic 2.5+
- **Testing**: pytest + pytest-asyncio + httpx

**Frontend**:
- **Framework**: React 18 + TypeScript 5.0+
- **Build Tool**: Vite 5.0
- **State Management**: Zustand or Redux Toolkit
- **UI Library**: Material-UI v5 or Ant Design
- **Routing**: React Router 6
- **Forms**: React Hook Form + Zod
- **API Client**: Axios
- **Testing**: Vitest + React Testing Library + Playwright

**Azure Services**:
- Azure App Service (Linux, P1v3)
- Azure Database for PostgreSQL Flexible Server
- Azure Blob Storage
- Azure Key Vault
- Azure Application Insights
- Azure Web Jobs
- Azure Event Grid (for engagement tracking)

### 16.3 Key Features Ported

✅ **Core Features**:
- Campaign management (draft, scheduled, running, queue-based)
- Subscriber management with JSONB attributes
- List management with segmentation
- Template system with Go template engine
- Media library (Azure Blob Storage)
- Import/export (CSV, JSON, ZIP)
- Public subscription forms and pages
- Campaign archive with RSS feed

✅ **Advanced Features**:
- Multi-SMTP server support (30+ servers)
- Queue-based email delivery with daily limits
- Time window enforcement (send only during configured hours)
- Smart Sending (prevent duplicate sends)
- Azure Event Grid integration (engagement tracking)
- Shopify purchase attribution
- Bounce handling (POP mailboxes + webhooks)
- Analytics and reporting
- Materialized views for performance
- RBAC with user and list roles
- OIDC authentication support
- Webhook logs for debugging

### 16.4 Success Criteria

The porting project will be considered successful if:

1. **Functionality**: 100% feature parity with existing listmonk
2. **Performance**: < 200ms API response time (p95)
3. **Reliability**: 99.9% uptime (excluding scheduled maintenance)
4. **Email Delivery**: > 100 emails/second processing rate
5. **Test Coverage**: 80%+ backend, 70%+ frontend
6. **Security**: Pass penetration testing audit
7. **Migration**: Zero data loss during migration
8. **User Satisfaction**: Positive feedback from beta testers

### 16.5 Next Steps

1. **Immediate** (Week 1-2):
   - Set up Azure resources (App Service, PostgreSQL, Blob Storage)
   - Create Git repository and project structure
   - Initialize FastAPI backend and React frontend scaffolds
   - Set up CI/CD pipelines

2. **Phase 1** (Week 3-8):
   - Implement database models (SQLAlchemy)
   - Create core API endpoints (campaigns, subscribers, lists)
   - Build basic React pages
   - Set up authentication

3. **Phase 2** (Week 9-16):
   - Implement campaign sending logic
   - Build template system
   - Add media management
   - Create analytics views

4. **Phase 3** (Week 17-24):
   - Implement queue system with Web Jobs
   - Add multi-SMTP support
   - Build bounce handling
   - Integrate Azure Event Grid

5. **Phase 4** (Week 25-32):
   - Complete Shopify integration
   - Add admin features (users, roles, settings)
   - Implement import/export
   - Build public pages

6. **Phase 5** (Week 33-40):
   - Comprehensive testing (unit, integration, E2E)
   - Performance optimization
   - Security audit
   - Bug fixes

7. **Phase 6** (Week 41-48):
   - Data migration preparation
   - Staging environment testing
   - Blue-green deployment setup
   - User acceptance testing

8. **Phase 7** (Week 49-56):
   - Production migration
   - Parallel running period
   - Final cutover
   - Post-launch monitoring and optimization

### 16.6 Contact and Support

For questions or clarifications about this porting plan:
- **Technical Lead**: [Your Name]
- **Project Manager**: [PM Name]
- **Repository**: [GitHub URL]
- **Documentation**: [Docs URL]

---

**Document Version**: 1.0
**Last Updated**: 2025-11-13
**Total Pages**: ~150 (estimated when printed)
**Total Lines**: 6,900+
**Status**: ✅ **COMPLETE**

---

This porting plan is a living document and should be updated as the project progresses. Regular reviews (every 2 weeks) are recommended to ensure alignment with project goals and timelines.

**Good luck with the port!** 🚀
