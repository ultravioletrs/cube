---
id: private-model-upload
title: Private Model Upload
sidebar_position: 3
---

## Uploading Private Models to Cube AI

This guide explains how to upload private models into the Ollama runtime inside a confidential VM.

## 1. Package Model Files

```bash
tar -czvf model-name.tar.gz /path/to/model/files
```

## 2. Transfer and Extract Model in CVM

```bash
scp model-name.tar.gz user@<cvm-ip>:~
gunzip model-name.tar.gz
tar -xvf model-name.tar
```

## 3. Copy Into Ollama

```bash
docker cp /path/to/extracted ollama:/models/
```
