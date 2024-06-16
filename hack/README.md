# Kor hack GuideLines

This document describes how you can use the scripts from [`hack`](.) directory
and gives a brief introduction and explanation of these scripts.

## Overview

The [`hack`](.) directory contains scripts that ensure continuous development of kor,
enhance the robustness of the code, improve development efficiency, etc.
The explanations and descriptions of these scripts are helpful for contributors.

## Key scripts

- [`find_exceptions.sh`](find_exceptions.sh): This script could be used to discover false-positive default resources in different K8s distributions. The output could be later merged into `pkg/kor/exceptions` for kor to ignore it in future releases.
