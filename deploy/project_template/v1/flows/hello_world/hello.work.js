import {
  hiphops,
  notify,
} from "https://deno.land/x/hiphops@2407-cromulent-fox/mod.ts";

hiphops.run(async ({ data }) => {
  // The notify service is built into hiphops.io, no integration required
  await notify.sendEmail({
    // Recipient addresses need to be added to your account's allow list before you
    // can send.
    to: ["someone@example.com"],
    // In this case, data is the raw pull_request event from GitHub (plus the `hops` metadata)
    // GitHub's own docs describe the exact structure of events
    content: `PR ${data["number"]} was merged!`,
  });
});
