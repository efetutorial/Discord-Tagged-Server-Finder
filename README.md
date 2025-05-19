# NOT WORKING HASH FUNC IS BROKEN

# Discord Server Tag Automation Script (GUILD_TAGS) - Go Version

Since around May 6th, 2025, newly created Discord servers have a small, random chance (approximately 1 in 1000) of including the experimental tag feature; this Go program automates the repetitive process of creating and deleting servers to help find one.

> [!NOTE]
> This project is a Go-based adaptation of the original JavaScript version found at [https://gist.github.com/bytexenon/db8e7dce72bb6a21aa2277de834de1d1](https://gist.github.com/bytexenon/db8e7dce72bb6a21aa2277de834de1d1). It has been modified to use direct Discord API calls with token authentication instead of relying on client-side functions.

> [!NOTE]
> Successfully obtaining a server with this flag is only the first step; actually *using* the tag functionality requires boosting the server 3 times


      
> [!CAUTION]
> ## 🚨 **EXTREME RISK & DIRECT TOS VIOLATION – READ IMMEDIATELY** 🚨
>
> ---
>
> ### **CRITICAL WARNING: YOUR ACCOUNT IS AT SUBSTANTIAL RISK!**
>
> Using **ANY** automated scripts, including the one detailed here, is a **FUNDAMENTAL AND EXPLICIT VIOLATION** of Discord's Terms of Service (ToS). By proceeding, you are knowingly engaging in activities that Discord strictly prohibits and actively monitors for.
>
> ### **THE CONSEQUENCES ARE SEVERE AND CAN BE IMMEDIATE:**
>
> *   **HIGH LIKELIHOOD OF ACCOUNT FLAGS:** Your account activity will likely be detected by Discord's anti-abuse systems.
> *   **TEMPORARY SUSPENSIONS:** You may lose access to your account for a period.
> *   **PERMANENT ACCOUNT TERMINATION (BANNING):** This is a very real possibility, resulting in the **IRREVERSIBLE LOSS** of your account, all associated servers you own, your messages, and your communities.
> *   **ACTION WITHOUT WARNING:** Discord is under no obligation to warn you before taking disciplinary action.
>
> ---
>
> ### **PROCEED WITH EXTREME CAUTION AND FULL ACKNOWLEDGEMENT OF THESE IMMINENT RISKS.**
> ### **PROCEED WITH EXTREME CAUTION AND FULL ACKNOWLEDGEMENT OF THESE IMMINENT RISKS.**
> ### **PROCEED WITH EXTREME CAUTION AND FULL ACKNOWLEDGEMENT OF THESE IMMINENT RISKS.**
> ---

> [!WARNING]
> ## **DISCLAIMER AND LIABILITY WAIVER**
>
> ---
>
> ### **BY USING THIS SOFTWARE, YOU ACKNOWLEDGE AND AGREE TO THE FOLLOWING:**
>
> *   **NO LIABILITY:** The author, contributors, and distributors of this software bear absolutely NO RESPONSIBILITY for any consequences resulting from your use of this tool.
> *   **USE AT YOUR OWN RISK:** Any actions taken with this software are done at your sole and exclusive risk. This includes but is not limited to:
>     * Account suspension or termination
>     * Loss of data or digital assets
>     * Any other penalties imposed by Discord
> *   **NO LEGAL RECOURSE:** You waive any right to seek damages or other compensation from the author(s) if your account is banned or penalized.
> *   **NO WARRANTY:** This software is provided "AS IS" without warranty of any kind, either expressed or implied.
> *   **EDUCATIONAL PURPOSE:** This tool is shared for educational purposes only to demonstrate API interactions.
>
> ### **YOU ARE SOLELY RESPONSIBLE FOR YOUR DECISION TO USE THIS SOFTWARE AND ALL CONSEQUENCES THEREOF.**
>
> ---

    

Before running the script, ensure you have completed the following step:

1.  **Disable 2FA (Recommended for smoother operation)**: While not strictly required for the script to function (as it doesn't interact with UI prompts), having 2FA disabled on the account might prevent potential friction or unexpected issues if Discord's API reacts to rapid server deletion from an account with 2FA. Re-enable 2FA after you finish using the script.

## Instructions

1.  **Download the Executable**: Obtain the `discord_tag_finder.exe` (or similarly named executable) file.
2.  **Run the Executable**: Double-click the executable file or run it from your terminal/command prompt.
    *   If running from the terminal, navigate to the directory where you saved the executable and type:
        ```shell
        .\discord_tag_finder.exe
        ```
3.  **Enter Token**: The script will prompt you to enter your Discord token. Paste your token and press Enter.

The script will then start creating and checking servers.

**Remember to re-enable 2FA on your account once the script has found a server or you decide to stop it.**

## token.txt
Create it so that each line contains a Discord user token:
```
token1
token2
token3
...
```

## proxy.txt
Create it so that each line contains a proxy address (e.g., http://user:password@ip:port or http://ip:port):
```
http://127.0.0.1:8080
http://kullanici:sifre@192.168.1.1:3128
...
```

Using a proxy when starting the program
