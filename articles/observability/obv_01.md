# observability from scratch, pt 1

> Debugging is twice as hard as writing the code in the first place.
>
Brian W. Kernighan

## why logging matters

### story: startup trouble

Suppose we're working for a "micro-mobility" startup. Our business works like this: people rent special e-bikes stationed in cities (called "host cities").

A user can rent the bike they'd like to use by scanning a QR code printed on the bike w/ their phone: the **front-end**.
This is then translated to a request to our backend **business-logic** server, which checks a few rules, like

- customer payment and auth
- local regulations
- whether this bike's in service-
The **business logic** server then forwards that request to a **bike-controller** server stationed near the user's city, which is in constant communication with the bikes over TCP-IP.

The **bike-controller** translates the **business logic** into  control packets sent over **tcp-ip**, which are sent to the bike's on-board **microcontroller**, which handles the rest.

One last wrinkle: the bike's **microcontroller** doesn't have TCP-IP, just bluetooth. So, in a time-honored tradition of ugly hacks, we screw a little cell phone radio with both TCP-IP and Bluetooth on to the bike and have it translate for us. (It uses a SIM card and everything.)

```mermaid
graph TD

routing
subgraph frontend
    I{{IOS}} 
    A{{ANDROID}} 
end
I == http ==> routing
A == http ==> routing  -- "(http)" --> logic
subgraph backend
routing((routing))
logic-. "check auth" .-> logic
logic-. "check payments" .-> logic
logic-. "check law" .-> logic

logic  == "(http): auth OK" ==> controller
end

subgraph embedded
controller == "tcp-ip" --> radio([radio]) == bluetooth ==> microcontroller([microcontroller]) ==> bike{bike}
end
```

One day at work, you get a call from the CTO: one of your most important clients is complaining that his bike "isn't working". You press for more information, and you get the bike's serial number: `00420-0069`, but that's pretty much it. **The CTO wants an answer by lunch**.

There are dozens of places this system could have failed.

- Was it the frontend?
  - If so, IOS or Android?
  - If so, what version of the frontend is he using?
- Was it the business logic?
  - What version of the business logic is running? Are multiple versions running at the same time in different regions? Which server and region did he get routed to?
  - What's his payment status? Did the auth go through?
  - Have the legal restrictions in his area changed recently?
  - Is the bike still authorized to work?
- Was it the bike controller?
  - If so, what model is his bike?
- When was the last time it worked? What was the state of the software system when it _did_ work?
- What model is his bike? How old is it?
- Maybe it's the radio?

It could take _ages_ to narrow down such a vague problem. But we do have really good logs, so let's take a peek and see what we can find:

```sh
# search using ripgrep for a line containing "error" and the serial number, then format the result using JQ
rg "error.+00420-0069" | jq 
```

We find a couple of errors, but the most recent one is from this morning:

```json
{
  "Msg": "unlock failed",
  "Level": "error",
  "Commit": "dcead05e453fbe3b8a3c290fd899c746670eabbc",
  "AppID": "bike-controller",
  "InstanceID": "d7dcc559-fa2f-4b64-a5b3-76bf230d6aff",
  "Error": "dial tcp: connection refused",
  "BikeSN": "00420-0069",
  "BikeModel": "8086C",
  "RadioType": "V1B",
  "RadioSN": "00360-NOSCOPE",
  "RadioIP": "5761:4219:e79e:808d:2701:3e0a:8654:3f33",
  "TraceID": "40743249-dcaa-4090-8638-7086a66a0173",
  "RequestIDs": [
    "4e5d4a6a-be0f-413b-80c4-ba45094e40a6", 
    "7da6c136-c467-4e63-aa25-ad7b15937aa3", 
    "4c60b6ee-9032-4a61-93d8-5a736161edde"
  ],
    "Time": "2023-02-10T06:22:22-7",
  "ControlPacket": {
    "Raw": "OGQyMmMyMGYtNzlkNi00MWY4LWI3MzItNDg1MTZiOWY3ZTU1",
    "Kind": "UNLOCK"
  }
}
```

We look for that TraceID to see how things went in the other services:

```sh
rg 40743249-dcaa-4090-8638-7086a66a0173
```

and we find messages like the following (shortened somewhat)

```json
{"Msg": "UNLOCK: to backend", "Level": "Debug", "AppID": "frontend-ios"...}
{"Msg": "got UNLOCK request from frontend", "Level": "Info", "AppID": "backend"}
{"Msg": "UNLOCK authorized", "Level": "Info", "AppID": "backend"}
{"Msg": "got UNLOCK request from backend", "Level": "Info", "AppID": "bike-controller"}
```

OK, so it's not the frontend or backend. and it doesn't seem like it's the bike controller, either. Probably not a software issue. In fact, it **probably is the radio**. Maybe it's lost power? Sometimes the connections get a bit loose.

You ask the local techs to take a look at the VIP's radio. The wiring's fine, but _someone took the SIM card_.  Pretty worrying for later, but we've cracked the case.

The difference between solving this problem being trivial and solving this problem being impossible is _clear, detailed, accessible logging_.

## log at service boundaries

Logging should tell you everything error messages don't.

- Log at handler level.

## design your API so that you can't fail to log

## always carry the following items in every log

**Application ID**. this should correspond to a single binary and never change over the life of an application.
- Git commit & release tag.  I've spent too much of my life tracking down bugs that literally don't exist on my version of the code.

```bash
    #!/bin/bash
    # set as environment variables
    export APP_COMMIT=$(git rev-parse HEAD)
    export APP_TAG=$(git --points-at head)
```

**InstanceID** This should be globally unique and generated at program startup. This lets you tell different instances of the same program from each other.

- **TraceID**.  Unique ID. Logs in a chain of events share a TraceID as it travels throughout your system. We'll talk more about tracing later.
- **RequestID**. If a TraceID identifies a 'chain' of events, a RequestID identifies a single link in that chain.

```go

#### don't log in the database call, log AFTER the database.


### everything is always broken

Software exists on a spectrum from "somewhat broken" to "completely broken".
Even the most commonly-used, battle-tested software can have lurking bugs for `decades`.

## our premise

When something goes wrong, we

- **Logging** - "what happened"?
- **Tracing** - "why led to this happening?"
- **Metrics** - "how does this usually happen?"
- **Metadata** - "what can we know about the place and time where it happened?"

## disambiguating metrics

Supposed we get a log message that looks like this:

> server:
