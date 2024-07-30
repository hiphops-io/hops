---
schedule: "*/5 * * * *"
# Set worker if the worker name is different to the md file name.
# worker: "foo" 
# Or to call a worker in a different flow:
# worker: "other_flow.bar"

# To instead trigger a flow worker in relation to an event (such as a PR), you can specify:
# on: "pull_request"

# Or to have a it triggered manually by users, create a command:
# command:
#   greeting: {type: "text", default: "Hello!", required: false}

# You can further filter events with an if expression.
# if: event.branch_name == "main"
# If expressions allow you to discard events in nanoseconds,
# meaning you can handle hyper noisy event sources
---

Hello runs every 5 minutes, sending an email saying "Hello"
