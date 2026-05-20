# Installing firebase in React Native

First, I have to install the base package.

```sh
npm install --save @react-native-firebase/app
```

To connect firebase to my Android app, I also had to follow the instructions
[here](https://rnfirebase.io/) up till **Step 3**.

## Lists

- Part 1: Setting up using React Native and Expo
- Part 2: Ditching Expo
- Part 3: Adding authentication with firebase

## Adding anonymous [authentication](https://rnfirebase.io/auth/usage)

To do this, I had to install the auth module.

```sh
npm install --save @react-native-firebase/auth
```

I've decided to sign people in anonymously. Because, I want to keep each
user's data separate on firestore, and it would very much help when creating
access control rules to have uniques UID generated for the users.

Also, if (or when?) a user creates a proper account, the anonymous account
can be linked to it.

So, I modified `App.js` to log the user in before displaying the homepage

```javascript
// App.js
import React from 'react';
import Icon from 'react-native-vector-icons/FontAwesome';

import { createMaterialBottomTabNavigator } from 'react-navigation-material-bottom-tabs';

import { createAppContainer } from 'react-navigation';
import { Provider as PaperProvider } from 'react-native-paper';
import { HomeScreen } from './views/home';

import auth from '@react-native-firebase/auth';

class App extends React.Component {
  constructor(props) {
    super(props);
    this.state = { user: null, initializing: true };
  }

  // Handle user state changes
  onAuthStateChanged(user) {
    this.setState({ user, initializing: false });
  }

  componentDidMount() {
    this.subscriber = auth().onAuthStateChanged(this.onAuthStateChanged.bind(this));
    auth().signInAnonymously();
  }

  componentWillUnmount() {
    this.subscriber();
  }

  render() {
    if (this.state.initializing) return null;
    return <PaperProvider><AppContainer /></PaperProvider>;
  }
}

const TabNavigator = createMaterialBottomTabNavigator({
  Home: { screen: HomeScreen },
});

const AppContainer = createAppContainer(TabNavigator);

export default App;
```
