# Gaming Group Manager
A tool to control gamming servers and programs while handling Member management. Written in Go. 

NOTE THIS CURRENTLY WILL ONLY WORK FULLY ON WINDOWS BASED MECHINES, CAN BE EDITED TO RUN ON ANY SYSTEM.

# Features

* Add/Remove Aditional Admins
* Add/Edit/Remove Aditional Games
* *  Icons Supported
* Add/Edit/Remove/Start/Restart/Stop Servers and Programs
* Change Panels Community Name
* Add/Edit/Remove Members
* * List Games
* * * Game Name with Icon
* * * Game Username
* * * Game Rank/Roll
* * * Game Departments
* * List Usernames
* * List Ranks
* * List Status (New, Active, Away, MIA, Inactive, Banned, Custom)
* * Admin Notes of user
* * List Forum URL
* * List Last Roll Call Date and Time
* Admin Notes Dashboard for all admins to comunicate
* Admin Log to keep track of whats being done
* Member Roll Call Page
* * Give Username
* * Optinal Resion for being Away
* Get User IP Page
* Emergency Backdoor Login (Can be removed)


UI is using Vali Admin and Font Awesome Icons.
Information is saved and hosted locally for images and in a SQL database for the rest.

# Images of Panel:

![Login Page](https://i.imgur.com/i0RNF9f.png)
![Dashboard](https://i.imgur.com/k9o5qfG.png)
![Member Info](https://i.imgur.com/iWxLyc9.png)
![Settings](https://i.imgur.com/3EH6mFk.png)
![Servers](https://i.imgur.com/mnp8hGc.png)
![Roll Call](https://i.imgur.com/6awkKjd.png)


# How to Use:

1. Download files
2. Upload database.sql to SQL Software
3. Edit main.go 
* Change Backdoor Logins
* Change Server Port
* Change SQL Information
* Change MD5Salt
4. Get and Compile Program
5. Run Program
6. Login with Backdoor and make new Admin account

NOTE: Make sure the "static" folder and files are in the same directory as the program.


# Packages Used:
* github.com/StackExchange/wmi
* github.com/gorilla/mux
*	github.com/gorilla/securecookie
* github.com/go-sql-driver/mysql

# Donations
<img src="https://blockchain.info/Resources/buttons/donate_64.png"/>
<p align="center">Please Donate To Bitcoin Address: <b>1AEbR1utjaYu3SGtBKZCLJMRR5RS7Bp7eE</b></p>
