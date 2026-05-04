import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind
} from "vscode-languageclient/node";

import {
  buildClientOptions,
  buildServerOptions,
  collectFixAllEdits,
  startupErrorMessage
} from "./wiring";

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  // Register commands first so they remain available even when the
  // server fails to start (the most useful one then is "Show Output
  // Channel" so the user can read the failure reason). Restart will
  // try a fresh start.
  context.subscriptions.push(
    vscode.commands.registerCommand("mdsmith.restartServer", () => restartServer(context)),
    vscode.commands.registerCommand("mdsmith.showOutput", () => showOutput())
  );

  // Wire fix-on-save once. The handler reads the setting on every
  // save so toggling the option does not require a restart.
  context.subscriptions.push(
    vscode.workspace.onWillSaveTextDocument((event) => {
      if (event.document.languageId !== "markdown") return;
      const fixOnSave = vscode.workspace.getConfiguration("mdsmith").get<boolean>("fixOnSave", false);
      if (!fixOnSave) return;
      event.waitUntil(
        vscode.commands.executeCommand(
          "vscode.executeCodeActionProvider",
          event.document.uri,
          new vscode.Range(0, 0, event.document.lineCount, 0),
          "source.fixAll.mdsmith"
        ).then(
          // collectFixAllEdits is typed against the structural
          // `TextEditLike` so wiring.ts stays decoupled from the
          // `vscode` runtime package; cast back to `vscode.TextEdit[]`
          // here because that's what `event.waitUntil` expects from a
          // willSave handler. The runtime objects are real
          // `vscode.TextEdit` instances forwarded from
          // executeCodeActionProvider, so the cast is safe.
          (actions) =>
            collectFixAllEdits(actions, event.document.uri) as vscode.TextEdit[]
        )
      );
    })
  );

  await startServer(context);
}

// startServer creates a fresh LanguageClient and start()s it. On
// failure it surfaces a quick-fix dialog (Download / Open Settings)
// without throwing, because the commands registered in activate()
// must remain usable so the user can retry.
async function startServer(_context: vscode.ExtensionContext): Promise<void> {
  const cfg = vscode.workspace.getConfiguration("mdsmith");
  const binary = cfg.get<string>("path", "mdsmith");
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

  const serverOptions: ServerOptions = buildServerOptions(
    binary,
    TransportKind.stdio,
    workspaceRoot
  );
  const clientOptions: LanguageClientOptions = buildClientOptions(
    vscode.workspace.createFileSystemWatcher("**/.mdsmith.yml")
  );

  client = new LanguageClient("mdsmith", "mdsmith", serverOptions, clientOptions);

  try {
    await client.start();
  } catch (err) {
    const choice = await vscode.window.showErrorMessage(
      startupErrorMessage(err),
      "Download mdsmith",
      "Open Settings",
      "Show Output"
    );
    if (choice === "Download mdsmith") {
      await vscode.env.openExternal(
        vscode.Uri.parse("https://github.com/jeduden/mdsmith/releases")
      );
    } else if (choice === "Open Settings") {
      await vscode.commands.executeCommand("workbench.action.openSettings", "mdsmith");
    } else if (choice === "Show Output") {
      showOutput();
    }
  }
}

// restartServer stops the running client (if any) and starts a
// fresh one. Useful when the user fixes `mdsmith.path`, rebuilds
// the binary, or otherwise wants to recover without reloading the
// VS Code window.
async function restartServer(context: vscode.ExtensionContext): Promise<void> {
  if (client) {
    try {
      await client.stop();
    } catch {
      // Ignore — a half-started client may refuse to stop, but
      // dropping the reference is enough to reclaim it.
    }
    client = undefined;
  }
  await startServer(context);
}

// showOutput reveals the "mdsmith" output channel where the
// language client logs RPC traffic and the server's stderr.
function showOutput(): void {
  // The LanguageClient registers an OutputChannel under
  // outputChannelName ("mdsmith"). Calling outputChannel.show on
  // the client's own handle is the safest way to reveal it without
  // importing internals.
  client?.outputChannel.show(true);
}

export async function deactivate(): Promise<void> {
  if (client) {
    await client.stop();
    client = undefined;
  }
}
