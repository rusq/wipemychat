# Wipe My Chat
Delete all your messages in public and private chats.

---
> _In loving memory of V. Gorban, 1967-2022._
---

## Demo

![](assets/wipemychat.gif)

## Usage

### GUI mode

1. Download the release from the [Releases page][1];
2. Unpack;
3. Run.

You will need:
- Telegram API ID
- Telegram API HASH

The program provides easy to follow instructions on how to get those.

To authenticate, you will use your Telegram Account phone number and the code,
that will be sent to you in-app or a text message (SMS).

### Batch/Script mode

You can also run the deletion from script.  Follow these steps:

1. Get the list of chat IDs that you want to wipe:
   ```shell
   wipemychat -list
   ```
2. Use the chat IDs with `-wipe` flag:
   ```shell
   wipemychat -wipe 12345,56789
   ```

### Logging out

If you need to log in under a different account (or phone number), you can
logout without deleting the application credentials by running:
```
wipemychat -logout
```

### Complete reset

If you need to completely reset the authentication, for example, if you
accidentally entered the wrong login details, or App Hash and App Secret, run:

```
wipemychat -reset
```

This deletes both files: session and application credentials. You will be asked
to authenticate again.

## Licence
GNU Public Licence 3.0, see [LICENCE][2]

[1]: https://github.com/rusq/wipemychat/releases
[2]: LICENCE
