const inquirer = require('inquirer');

const run = async () => {
  const { name } = await askName();
  while (true) {
    const answers = await askChat();
    const { message } = answers;
    console.log(`${name}: `, message);
  }
};

const askChat = () => {
  const questions = [
    {
      name: "message",
      type: "input",
      message: "Enter chat message:"
    }
  ];
  return inquirer.prompt(questions);
};

const askName = () => {
  const questions = [
    {
      name: "name",
      type: "input",
      message: "Enter your name:"
    }
  ];
  return inquirer.prompt(questions);
};

run();
