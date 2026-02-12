<div align="center">
	<img src="./images/logo.png" width="400" height="400" alt="CloudDrop" style="border-radius: 50%;"><br>
	<h1>CloudDrop</h1>
</div>
The idea behind this project are to create a simple CLI tool that allows you to upload files to a cloud storage provider or share files P2P(peer to peer) in a local network

### Have you ever wanted to share a file with someone or wanted to send a file from your computer to your mobile phone?
1. Cloud based approach:
Well, this is the tool for you! Why use the middleman of Discord, Gmail, Slack, or any other service when you can use a simple script that does not store your data. Take care of your security and privacy by using this script entirely for free.
<strong>
It runs locally on YOUR machine and is connected to YOUR own cloud provider.
</strong>

2. P2P approach:
Share files with your friends or family without the need for a middleman. The only requirement is that both of the clients must have the clouddrop cli installed.
<strong>
Pure byte-to-byte P2P sharing, no middleman needed. FULL SECURITY AND PRIVACY.
</strong>

### Description
This is a script that allows you to upload files to a cloud storage provider, and have them be available for sharing using a link. The files are stored in a cloud provider of choice(Azure, Google Cloud, AWS), the program itself is written in Go and is a CLI tool that listens for file uploads in a selected 'drop' directory on your machine.

### Setup and Installation
To do

### Usage
<strong>Peer to Peer on local network</strong>
Client A (Sender):
````bash
clouddrop drop /path/to/file
````
Client B (Receiver):
````bash
clouddrop pick
````

<strong>Over public internet</strong>
Client A (Sender):
````bash
clouddrop send /path/to/file
````
Client B (Receiver):
````bash
clouddrop receive <code>
````

#### Commands
| Command | Description |
| --- | --- |
| drop | Upload files to P2P network. Must provide a valid file path to a file or directory. |
| pick | Receive files from P2P network. No arguments required. |
| send | Upload files over public internet. Must provide a valid file path. |
| receive | Receive files over public internet. Must provide a valid code received from the sender. |

### Example use cases:
1. Share files between computers on your local network (P2P)
2. Share files over the public internet without cloud storage
3. Share files on your machine, effectively copying them to the receiver's path

<strong>Cloud based approach</strong>
To do
### Contributing
If you want to contribute to this project, you can do so by forking the repository and making a pull request.
