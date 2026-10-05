; ZarvandVPN Windows Installer (Inno Setup) - supports Windows 7 through 11
; Build: iscc /DMyAppVersion=1.0.2 /DArch=x64 installer.iss

#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif
#ifndef Arch
  #define Arch "x64"
#endif

[Setup]
AppId={{8E5B3D2A-6C41-4F0E-9A77-ZARVANDVPN01}
AppName=ZarvandVPN
AppVersion={#MyAppVersion}
AppPublisher=ZarvandVPN
DefaultDirName={pf}\ZarvandVPN
DefaultGroupName=ZarvandVPN
DisableProgramGroupPage=yes
OutputBaseFilename=ZarvandVPN-{#MyAppVersion}-{#Arch}-setup
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern
UninstallDisplayIcon={app}\ZarvandVPN.exe
MinVersion=6.1sp1
ArchitecturesAllowed={#Arch}compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\build\ZarvandVPN-{#MyAppVersion}-{#Arch}.exe"; DestDir: "{app}"; DestName: "ZarvandVPN.exe"; Flags: ignoreversion
Source: "..\assets\zarvand.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\ZarvandVPN"; Filename: "{app}\ZarvandVPN.exe"; IconFilename: "{app}\zarvand.ico"
Name: "{group}\{cm:UninstallProgram,ZarvandVPN}"; Filename: "{uninstallexe}"
Name: "{userdesktop}\ZarvandVPN"; Filename: "{app}\ZarvandVPN.exe"; IconFilename: "{app}\zarvand.ico"; Tasks: desktopicon

[Run]
Filename: "{app}\ZarvandVPN.exe"; Description: "{cm:LaunchProgram,ZarvandVPN}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; leave user settings in %APPDATA%\ZarvandVPN on uninstall (config + extracted core)
