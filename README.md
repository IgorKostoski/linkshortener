# Geo-Distributed Link Shortener & Analytics Platform

A robust, scalable link shortening service built with Go, leveraging a modern DevOps stack including Docker, Ansible,
Prometheus, Grafana, and GitHub Actions for CI/CD and SonarCloud for code quality. This project serves as a
comprehensive demonstration of building, deploying, and monitoring a web application using industry best practices.

[![Go CI and Docker Build](https://github.com/IgorKostoski/linkshortener/actions/workflows/ci.yml/badge.svg)](https://github.com/IgorKostoski/linkshortener/actions/workflows/ci.yml)
[![SonarCloud Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=IgorKostoski_linkshortener&metric=alert_status)](https://sonarcloud.io/summary/overall?id=IgorKostoski_linkshortener)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=IgorKostoski_linkshortener&metric=coverage)](https://sonarcloud.io/summary/overall?id=IgorKostoski_linkshortener)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=IgorKostoski_linkshortener&metric=bugs)](https://sonarcloud.io/summary/overall?id=IgorKostoski_linkshortener)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=IgorKostoski_linkshortener&metric=vulnerabilities)](https://sonarcloud.io/summary/overall?id=IgorKostoski_linkshortener)

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Project Architecture](#project-architecture)
- [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Local Development Setup](#local-development-setup)
- [Application Endpoints](#application-endpoints)
- [CI/CD Pipeline](#cicd-pipeline)
- [Deployment (Ansible)](#deployment-ansible)
    - [Target Server Setup (UTM VM)](#target-server-setup-utm-vm)
    - [Running Ansible Playbooks](#running-ansible-playbooks)
- [Monitoring](#monitoring)
- [Code Quality](#code-quality)
- [Future Enhancements & DevOps Practices Explored](#future-enhancements--devops-practices-explored)
- [License](#license)

## Features

* **URL Shortening:** Converts long URLs into short, manageable links.
* **Redirection:** Seamlessly redirects users from short links to their original destinations.
* **Basic Analytics (Conceptual):** Foundation laid for tracking link clicks (currently Prometheus metrics track
  creations/redirects).
* **RESTful API:** Simple API for creating short links.
* **Persistent Storage:** Uses PostgreSQL to store URL mappings.
* **Containerized Deployment:** All services are Dockerized for consistency and scalability.
* **Automated CI/CD:** GitHub Actions for testing, analysis, building, and (simulated) deployment.
* **Infrastructure as Code (IaC) Principles:** Ansible for server configuration and application deployment.
* **Monitoring:** Prometheus for metrics collection and Grafana for visualization.
* **Code Quality Assurance:** SonarCloud integration for static analysis.

## Tech Stack

* **Backend:** Go (Golang)
* **Database:** PostgreSQL
* **Containerization:** Docker, Docker Compose
* **CI/CD:** GitHub Actions
* **Configuration Management & Deployment:** Ansible
* **Monitoring:**
    * Prometheus (Metrics Collection)
    * Grafana (Visualization)
* **Code Quality:** SonarCloud
* **Version Control:** Git, GitHub
* **Target OS for Deployment:** AlmaLinux 9 (arm64) - RPM-based Linux
* **Local VM for Server:** UTM with QEMU (for Apple Silicon)

## Project Architecture

```mermaid
graph TD
    User -->|1. POST /shorten (long URL)| GoApp
    GoApp -->|2. Stores mapping| PostgreSQL
    GoApp -->|3. Returns short URL| User

    User -->|4. GET /{short_code}| GoApp
    GoApp -->|5. Retrieves long URL| PostgreSQL
    GoApp -->|6. Redirects (302)| OriginalWebsite
    GoApp -->|7. Records redirect (metrics)| Prometheus

    CI_CD_Pipeline[GitHub Actions] -->|Builds & Pushes Image| GHCR[GitHub Container Registry]
    CI_CD_Pipeline -->|Triggers (Simulated)| Ansible
    Ansible -->|Deploys Stack| UTM_VM[UTM VM (AlmaLinux)]

    UTM_VM -->|Contains| DockerizedServices[Docker: GoApp, PostgreSQL, Prometheus, Grafana]
    Prometheus -->|Scrapes /metrics| GoApp
    Grafana -->|Queries| Prometheus
    SonarCloud -->|Analyzes| GitHubRepo[GitHub Repository]
    CI_CD_Pipeline -->|Sends report| SonarCloud