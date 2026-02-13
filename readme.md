<div align="center">
	<img src="./images/logo.png" width="400" height="400" alt="CloudDrop" style="border-radius: 50%;"><br>
	<h1>CloudDrop</h1>
</div>
The idea behind this project is to create a simple CLI tool that allows you to upload files via public internet to a connected client or share files P2P(peer to peer) on a local network.

### Have you ever wanted to share a file with someone? I bet you have!
1. Cloud based approach:
Why use the middleman of Discord, Gmail, Slack, or any other service when you can use a simple cli that does not store your data. Take care of your security and privacy by using this tool entirely for free.
<strong>
It runs locally on YOUR machine and is connected to a cloud storage provider which handles the encryption and security of your files.
</strong>

2. P2P approach:
Share files with your friends or family without the need for a middleman. The only requirement is that both of the clients must have the clouddrop cli installed.
<strong>
Pure byte-to-byte P2P sharing, no middleman needed. FULL SECURITY AND PRIVACY.
</strong>

### Description
Clouddrop enables you to share files or directories either P2P or over the public internet. It uses kafka and google storage. It ensures high security via access codes.

### Setup and Installation
To do

### Usage
<strong>Peer to Peer on local network</strong><br>
Client A (Sender):
````bash
clouddrop drop /path/to/file
````
Client B (Receiver):
````bash
clouddrop pick
````

<strong>Over public internet</strong><br>
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
1. Share files or directories between computers on your local network (P2P).
2. Share files over the public internet.
3. Share files on your machine, effectively copying them to the receiver's path.

### Contributing
If you want to contribute to this project, you can do so by forking the repository and making a pull request.
Please be advised that for development purposes, you will need to have an active credentials.json file in the root directory of the project and have an available bucket name in google cloud storage.
