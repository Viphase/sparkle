# Sparkle Product Spec

## One-Sentence Description

Sparkle is a local-first writing/project manager that turns rough ideas into structured, trackable project workspaces through a beautiful Go terminal UI.

## Target User

Primary v1 user:
- one person
- manages personal writing/software/project ideas
- wants structure without heavyweight project-management software
- works locally
- values fast keyboard workflow and pleasant UX

Future users:
- small teams
- collaborators
- shared workspaces

Do not optimize v1 for teams.

## Core Concepts

### Spark

A spark is a short undeveloped idea.

It should be quick to capture and easy to later develop.

Examples:
- "Go TUI writing manager with AI"
- "CLI for organizing startup ideas"
- "Novel tracker with daily consistency chart"

### Project

A project is a developed workspace created directly or promoted from a spark.

A project contains:
- title
- description
- architecture
- target audience
- GitHub page/link
- roadmap
- milestones
- notes
- tracker data

## Product Flow

1. User opens Sparkle.
2. Dashboard shows workspaces, stats, recent activity, and active projects.
3. User captures a spark quickly.
4. Later, user opens Sparks Bubble.
5. User selects a spark and promotes it into a project.
6. Sparkle creates project Markdown files.
7. User fills project sections manually or with future AI guidance.
8. Tracker automatically detects activity and shows consistency graphs.

## Main Screens

### Dashboard

Purpose:
- orientation
- daily overview
- stats
- quick access

Show:
- spark count
- active project count
- recent projects
- today’s work stats
- weekly consistency chart
- next milestones
- shortcuts

Workspace switching lives in Settings — v1 assumes one active workspace at a time.

### Sparks Bubble

Purpose:
- capture and develop short ideas

Features:
- create spark
- edit spark
- archive spark
- search/filter
- promote spark to project
- enter future AI-guided development flow

### Project Workspace

Purpose:
- structured work on developed projects

Features:
- project list
- project detail
- editable fields
- architecture section
- target audience section
- GitHub link section
- roadmap
- notes
- tracker link
- AI guide link

### Tracker

Purpose:
- show consistency and momentum

Features:
- daily consistency chart
- weekly activity chart
- word count trend
- project velocity
- streak
- session time
- milestones
- task status

### AI Guide

Purpose:
- future AI mentor flow

v1 should include:
- provider interface
- mock provider
- prompt builder
- basic AI screen

Real API integration comes later.

## v1 Product Boundaries

Include:
- local workspaces
- sparks
- projects
- Markdown storage
- automatic tracking basics
- charts
- themes
- mock AI interface

Exclude:
- cloud sync
- team accounts
- real-time collaboration
- plugin marketplace
- Pomodoro timer
- AI fine-tuning
- complex permissions
