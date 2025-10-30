# AI Agent Configuration

This directory contains specialized AI agent prompts and multi-agent workflow definitions for the Joint Analysis project.

## Directory Structure

### Individual Agent Prompts (`/agents/`)
- `backend-agent.md` - Golang backend development specialist
- `frontend-agent.md` - React/TypeScript frontend specialist
- `testing-agent.md` - Testing and QA specialist
- `review-agent.md` - Code review and quality assurance specialist

### Multi-Agent Workflows (`/agents/workflows/`)
- `feature-development.md` - End-to-end feature development workflow
- `bug-fix-workflow.md` - Bug investigation and resolution workflow
- `release-workflow.md` - Release preparation and deployment workflow

## Usage Guidelines

### Single Agent Usage
Each agent prompt file contains specialized instructions for a specific development role. Use these when you need focused expertise in a particular area.

### Multi-Agent Workflows
Workflow files define how multiple specialized agents should collaborate on complex tasks. They include:
- Agent roles and responsibilities
- Handoff procedures between agents
- Quality gates and checkpoints
- Communication protocols

## Agent Principles

All agents follow the core principles defined in `development-guidelines.md`:
- **Test-Driven Development (TDD)** - Red, Green, Refactor cycle
- **Proper Error Handling** - Explicit error checking and validation
- **Code Quality** - Linting, formatting, and documentation standards
- **Commit Standards** - Atomic commits with clear messages

## Creating New Agents

When creating new agent prompts:
1. Follow the established naming convention: `{role}-agent.md`
2. Include role definition, responsibilities, and specific guidelines
3. Reference the core development principles
4. Add entry to this README
5. Update `development-guidelines.md` if needed

## Creating New Workflows

When defining multi-agent workflows:
1. Use descriptive names: `{task-type}-workflow.md`
2. Define clear agent roles and handoffs
3. Include quality checkpoints
4. Specify success criteria
5. Document the workflow in this README