### Scheduled messages

**Need to send a message later?** Use the `/schedule` command -- it will be sent automatically at the time you choose.

**How to schedule:**

Switch to the channel or direct message where you want the message to appear, then type:

`/schedule at <time> [on <date>] message <your message text>`

*   Replace `<time>` with the send time (e.g., `at 9:00AM`, `at 17:30`, `at 3pm`). Your timezone setting in Mattermost is used.
*   Optionally, use `on <date>` to specify a date. Replace `<date>` with the date in any of these formats:
    * `YYYY-MM-DD`: e.g. `on 2026-01-15`
    * `Day of week`: e.g. `on mon` or `on Monday`
    * `Short day of month`: e.g. `on 3jan` or `on 26dec`
    * If you skip the date, or use `Day of week` or `Short day of month` format, it schedules for the soonest possible day/time in the future that matches (e.g. today/tomorrow for no date, this Wednesday or next Wednesday for `wed`, this June 3rd or June 3rd next year for `3jun`, etc.
*   Replace `<your message text>` with your actual message.

**Examples:**

*   To schedule a sales meeting for 2:15PM:
    ```
    /schedule at 2:15PM message Sales meeting now
    ```
*   To schedule a Christmas greeting in the morning:
    ```
    /schedule at 9am on 25dec message Merry Christmas!
    ```
*   To schedule a Friday afternoon coffee break:
    ```
    /schedule at 3pm on fri message Coffee break
    ```
*   To schedule something in the far future:
    ```
    /schedule at 13:00 on 2050-01-01 message End of the world
    ```

**See your scheduled messages:** `/schedule list`

**Send scheduled messages now:** List your messages, click the `Send` button below the message.

**Delete scheduled messages:** List your messages, click the `Delete` button below the message.

**Get help:** `/schedule help` (Shows this information again).
