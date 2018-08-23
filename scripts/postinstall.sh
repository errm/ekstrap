#!/bin/sh

systemctl daemon-reload > /dev/null || true
systemctl enable ekstrap.service > /dev/null || true
