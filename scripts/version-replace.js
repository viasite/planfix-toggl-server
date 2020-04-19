// заменяет строку "{{version}}" в README.md на новую версию
const fs = require('fs');
const packageJson = require('../package.json');

const str = fs.readFileSync('README.md', 'utf8');
const replaced = str.replace(/\{\{version\}\}/g, packageJson.version);
fs.writeFileSync('README.md', replaced, 'utf8');
