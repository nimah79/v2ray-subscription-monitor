; Inno Setup 6 — visual installer with modern wizard (https://jrsoftware.org/isinfo.php)
; Built from repo root via: ISCC /DBuildArch=amd64|arm64 /DMyAppVersion=x.y.z installer\windows\setup.iss
; For arm64 also pass: /DArm64=1

#ifndef MyAppName
#define MyAppName "V2Ray Subscription Monitor"
#endif
#ifndef MyAppVersion
#define MyAppVersion "0.0.1"
#endif
#ifndef BuildArch
#define BuildArch "amd64"
#endif

[Setup]
AppId={{C4E8F1B2-9D7A-5E3C-8F10-123456789ABC}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher=Open Source
AppPublisherURL=https://github.com/
WizardStyle=modern
PrivilegesRequired=lowest
DefaultDirName={localappdata}\Programs\{#MyAppName}
DisableProgramGroupPage=yes
CloseApplications=no
OutputDir=..\..\dist
OutputBaseFilename=v2ray-subscription-monitor-windows-{#BuildArch}-setup
UninstallDisplayIcon={app}\v2ray-subscription-monitor.exe
Compression=lzma2
SolidCompression=yes
#ifdef Arm64
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64
#endif

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\..\dist\v2ray-subscription-monitor-windows-{#BuildArch}.exe"; DestDir: "{app}"; DestName: "v2ray-subscription-monitor.exe"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\v2ray-subscription-monitor.exe"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\v2ray-subscription-monitor.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\v2ray-subscription-monitor.exe"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent
