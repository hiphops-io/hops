import {
  hiphops,
  notify,
} from "https://deno.land/x/hiphops@2407-cromulent-fox/mod.ts";

// data and subject don't need to be specified e.g. you could use:
// hiphops.run(async () => {...}) instead
// data contains the event payload, subject is the subject the message was received on.
hiphops.run(async ({ data, subject }) => {
  // The notify service is built into hiphops.io, no integration required
  await notify.sendEmail({
    // Recipient addresses need to be added to your account's allow list before you
    // can send.
    to: ["someone@example.com"],
    // In this case, data is the raw pull_request event from GitHub (plus the `hops` metadata)
    // GitHub's own docs describe the exact structure of events
    content: `Hello!`,
  });
});
