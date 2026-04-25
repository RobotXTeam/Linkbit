# Linkbit API Skeleton

All authenticated endpoints require `X-Linkbit-API-Key`. In production this key must only be sent over HTTPS.

## Public

`GET /healthz`

Returns controller process health.

`POST /api/v1/devices/register`

Registers a device with a one-time or reusable enrollment key.

Required JSON fields:

- `enrollmentKey`
- `name`
- `publicKey`

Returns the assigned device, virtual IP, and known relay nodes.

`GET /api/v1/devices/{id}/network-config`

Device-token authenticated endpoint that returns the device, allowed peers, policies, and relays needed by the agent to configure WireGuard.

Required header:

- `X-Linkbit-Device-Token`

## Overview

`GET /api/v1/overview`

Returns dashboard counters for devices, relay nodes, policies, and network health.

## Users And Groups

`POST /api/v1/users`

Creates a user.

`GET /api/v1/users`

Lists users.

`POST /api/v1/groups`

Creates a device group.

`GET /api/v1/groups`

Lists device groups.

## Relay Registry

`GET /api/v1/derp-map`

Returns a Tailscale-compatible `tailcfg.DERPMap` generated from registered relay nodes.

`POST /api/v1/relays/register`

Registers a relay node with controller-managed metadata.
Accepts `admin` or `relay` API key scope.

Required JSON fields:

- `id`
- `publicUrl`

`POST /api/v1/relays/heartbeat`

Refreshes relay liveness and load metrics.
Accepts `admin` or `relay` API key scope.

Required JSON fields:

- `id`

`GET /api/v1/relays`

Returns registered relay nodes.

`DELETE /api/v1/relays/{id}`

Removes a relay node from the controller registry.

## Device Registry

`GET /api/v1/devices`

Returns registered devices. Device enrollment will be added behind invitation keys in the next implementation step.

## Invitations

`POST /api/v1/invitations`

Creates an enrollment key. The plaintext token is returned only in the create response and only the HMAC digest is stored.

Required JSON fields:

- `userId`

Optional JSON fields:

- `groupId`
- `reusable`
- `expiresInSeconds`

## Network Policies

`POST /api/v1/policies`

Creates a network policy edge from source device/group to target device/group.

Required JSON fields:

- `id`
- `sourceId`
- `targetId`

`GET /api/v1/policies`

Returns network policies.
