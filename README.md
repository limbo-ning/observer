This system is the used for collecting and displaying data streams uploaded by HJT-2005/2017 protocol, and it mainly serves as the platform of internet based environment pollution detection devices nysqetworks.
The backend server is implemented in GoLANG, using mysql database.
This is designed to support thousands of concurrent TCP connections to upload real time data feeds, and deployed in practice with one thousand connections.

The frontend parts consists of two main components: the website implemented with React/Typescript/TailwindCSS, which is for the purpose of administration, data display, data export, and the other to be the counterpart implemented in Wechat APP, which is for the purpose of mobile display and subscription/alerts.
The frontend part is not uploaded here because it contains confidential data of the clients, while the backend implementation remains the same for different clients and therefore is safe for public repository.

