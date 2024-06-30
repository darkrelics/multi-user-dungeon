# Multi-User Dungeon Engine

The goal of this project is to create a commercial-quality multi-user dungeon (MUD) engine that is flexible enough to be used as either a conventional MUD or an interactive fiction game.

The current implementation includes an SSH server for secure authentication and communication between the player and the server. The engine is primarily written in Go. Additionally, there is a user management system stub written in JavaScript and various utility scripts written in Python.

## Current Objectives

- [x] Create the TCP server for client connections.
- [x] Create a text parser for user input.
- [x] Implement a player authentication system.
- [x] Implement a database for the game.
- [x] Implement a character creation system.
- [x] Implement a text colorization system.
- [ ] Construct the item system.
- [ ] Develop game mechanics.
- [ ] Design an ecenomic framework.
- [ ] Implement a world creation system.
- [ ] Develop simple Non-Player Characters (NPCs).
- [ ] Design a quest system.
- [ ] Build a direct messaging system.
- [ ] Create AI-controlled NPCs.
- [ ] Implement a dynamic content updating system.
- [ ] Build an interactive password change system.

## TODO

- [x] Fix output formatting for the client.
- [x] Allow players to enter their name.
- [x] Display the incoming IP address and Port on the server.
- [x] Add a help command.
- [x] Add a character list command.
- [x] Allow users to change their passwords.
- [x] Expand the character creation process.
- [x] Add take item command.
- [x] Add inventory command.
- [x] Add drop item command.
- [ ] Add a Message of the Day (MOTD) command.
- [ ] Implement Persistent Logging.
- [ ] Add the ability to delete characters.
- [ ] Add the ability to delete accounts.
- [ ] Implement an obscenity filter.
- [ ] Validate graph of loaded rooms and exits.
- [ ] Load item prototypes at start.
- [ ] Create function for creating items from prototypes.
- [ ] Add wear item command.
- [ ] Add remove item command.
- [ ] Add look at item command.
- [ ] Add examine item command.

## Deployment

Deploying the server involves several steps, from setting up your environment to running the server. Follow these steps to ensure a smooth deployment process:

1. **Install Go**: The server is written in Go, so you need to have Go installed on your system. Download it from the [Go website](https://golang.org/).

2. **Set Up AWS Account**: An AWS account is required for deploying certain components of the server, such as the authentication system. Sign up for an account [here](https://aws.amazon.com/) if you don't already have one.

3. **Configure AWS Credentials**: Ensure you have AWS credentials configured on your machine. These credentials should have sufficient permissions to create a Cognito user pool and the necessary IAM policies and roles. You can configure your credentials by using the AWS CLI and running `aws configure`.

4. **Deploy Cognito and IAM Resources**:

   - Navigate to the `scripts` directory within the project.
   - Run the `deploy_cognito.py` script using the command `python deploy_cognito.py`. This script will create the Cognito instance along with the required IAM policies and roles. It will also generate the `config.json` file needed to run the server. Ensure you have Python installed on your machine to execute this script.

5. **Install Go Dependencies**: Before starting the server, you need to install the necessary Go dependencies. In the root directory of the project, run `go mod download` to fetch all required packages.

6. **Initalize the Database**: The server uses a BoltDB database for the world data. You can initialize the database by running the `data_loader.go` script located in the `database` directory. Run the script using the command `go run .`. The output will be the `test_data.bolt` file, which contains the initial world data. Copy this file to the `mud` directory.

7. **Start the Server**: Finally, you can start the server by running `go run .` from the root directory of the project. This command compiles and runs the Go application, starting up your MUD server.

Ensure all steps are completed without errors before trying to connect to the server. If you encounter any issues during deployment, refer to the specific tool's documentation for troubleshooting advice.

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.
