import {
  EthereumClient,
  w3mConnectors,
  w3mProvider,
  WagmiCore,
  WagmiCoreChains,
  WagmiCoreConnectors,
} from "https://unpkg.com/@web3modal/ethereum@2.6.2";

import { Web3Modal } from "https://unpkg.com/@web3modal/html@2.6.2";

// 0. Import wagmi dependencies
const { mainnet, polygon, polygonMumbai, avalanche, arbitrum } =
  WagmiCoreChains;
const {
  configureChains,
  createConfig,
  watchAccount,
  watchSigner,
  disconnect,
  signMessage,
  getAccount,
} = WagmiCore;

//Define chains
const chains = [mainnet, polygon, polygonMumbai];
const projectId = "3d6930929763b3142513d912505eba46";

//Configure wagmi client
const { publicClient } = configureChains(chains, [w3mProvider({ projectId })]);

const wagmiConfig = createConfig({
  autoConnect: true,
  connectors: [
      ...w3mConnectors({ chains, version: 2, projectId }),
      new WagmiCoreConnectors.CoinbaseWalletConnector({
          chains,
          options: {
              appName: "html wagmi example",
          },
      }),
  ],
  publicClient,
});

//Create ethereum and modal clients
const ethereumClient = new EthereumClient(wagmiConfig, chains);
export const web3Modal = new Web3Modal(
  {
      projectId,
      walletImages: {
          safe: "https://pbs.twimg.com/profile_images/1566773491764023297/IvmCdGnM_400x400.jpg",
      },
      enableAccountView: true,
      enableExplorer: true,
      themeMode: "dark",
      themeVariables: {
          "--w3m-accent-color": "#FF8700",
          "--w3m-accent-fill-color": "#000000",
          "--w3m-background-color": "#000000",
          // '--w3m-background-image-url': '/images/customisation/background.png',
          // '--w3m-logo-image-url': '/images/customisation/logo.png',
          "--w3m-background-border-radius": "0px",
          "--w3m-container-border-radius": "0px",
          "--w3m-wallet-icon-border-radius": "0px",
          "--w3m-wallet-icon-large-border-radius": "0px",
          "--w3m-input-border-radius": "0px",
          "--w3m-button-border-radius": "0px",
          "--w3m-secondary-button-border-radius": "0px",
          "--w3m-notification-border-radius": "0px",
          "--w3m-icon-button-border-radius": "0px",
          "--w3m-button-hover-highlight-border-radius": "0px",
          "--w3m-font-family": "monospace",
      },
  },
  ethereumClient
);

//Variables
let challengeUrl;
let authId;
let verifyUrl;

//Utils
function displayError(msg) {
  const errorBox = document.getElementById("web3-error");
  errorBox.style.display = "block";
  errorBox.textContent = msg;
}

async function postJson(url, obj) {
  return fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(obj),
  });
}

//callbacks
async function onAccountConnected() {
  try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      const account = await getAccount();
      const challengeResp = await postJson(challengeUrl, {
          address: account.address,
          state: authId,
      });
      const challengeBody = await challengeResp.json();
      const signature = await signMessage({
          message: challengeBody.nonce,
      });
      //Catch denied error here and fail gracefully.
      console.log(signature);
      const verifyResp = await postJson(verifyUrl, {
          signed: signature,
          state: authId,
      });
      if (verifyResp.ok) {
          const verifyBody = await verifyResp.json();
          window.location.replace(verifyBody.redirect);
      } else {
          const verifyBody = await verifyResp.json();
          displayError("Verification error: " + verifyBody.message);
      }
  } catch (err) {
      console.error(err);
      displayError(err.message);
  }
}

async function onClick() {
  try {
      //check if we are already connected
      const account = await getAccount();
      if (
          account.address != undefined &&
          account.isConnected &&
          !account.isConnecting
      ) {
          await onAccountConnected();
      } else {
          await web3Modal.openModal();
      }
  } catch (err) {
      console.error(err);
      await onReset();
  }
}

async function onReset() {
  try {
      await disconnect();
      await web3Modal.closeModal();
  } catch (err) {
      console.error(err);
  }
}

watchAccount(async (account) => {
  if (
      account.address != undefined &&
      account.isConnected &&
      !account.isConnecting
  ) {
      await onAccountConnected();
  }
});

function init() {
  challengeUrl = document.getElementById("tmpl-challenge-url").value;
  authId = document.getElementById("tmpl-auth-id").value;
  verifyUrl = document.getElementById("tmpl-verify-url").value;

  const connectButton = document.getElementById("connect-button");
  connectButton.addEventListener("click", onClick);

  const resetButton = document.getElementById("reset-button");
  resetButton.addEventListener("click", onReset);


}

window.addEventListener("load", init);