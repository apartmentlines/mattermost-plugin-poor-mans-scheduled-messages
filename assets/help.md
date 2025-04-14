### Scheduled messages

**Need to send a message later?** Use the `/schedule` command -- it will be sent automatically at the time you choose.

**How to schedule:**

Switch to the channel or direct message where you want the message to appear, then type:

`/schedule at <time> [on <date>] message <your message text>`

*   Replace `<time>` with the send time (e.g., `at 9:00AM`, `at 17:30`, `at 3pm`). Your timezone setting in Mattermost is used.
*   Optionally, replace `[on <date>]` with the send date in `YYYY-MM-DD` format (e.g., `on 2025-01-15`). If you skip the date, it schedules for the soonest possible time (today or tomorrow).
*   Replace `<your message text>` with your actual message.

**Examples:**

*   To schedule 'Remember the team meeting!' for 2:15PM today/tomorrow:
    ```
    /schedule at 2:15PM message Remember the team meeting!
    ```
*   To schedule 'Merry Christmas!' on Christmas morning:
    ```
    /schedule at 9am on 2026-12-25 message Merry Christmas!
    ```

**See your scheduled messages:** `/schedule list`

**Delete scheduled messages:** List your messages, click the `Delete` button below the message.

**Get help:** `/schedule help` (Shows this information again).
