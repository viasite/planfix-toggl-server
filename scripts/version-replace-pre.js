// заменяет старую версию в README.md на строку "{{version}}"
const fs = require('fs');
const packageJson = require('../package.json');

const str = fs.readFileSync('README.md', 'utf8');
const regex = new RegExp(packageJson.version.replace(/\./g, '\\.'), 'g');
const replaced = str.replace(regex, '{{version}}');
fs.writeFileSync('README.md', replaced, 'utf8');
