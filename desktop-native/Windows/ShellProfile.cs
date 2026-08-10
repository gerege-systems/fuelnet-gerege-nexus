namespace GeregeNexusNativeWin;

public static class ShellProfile
{
#if KIOSK
    public const string FormFactor = "kiosk";
    public const string StartRoute = "/kiosk";
    public static readonly string[] Capabilities = ["escpos", "scanner", "serial", "device.identity", "kiosk.lockdown", "telemetry"];
#elif POS
    public const string FormFactor = "pos";
    public const string StartRoute = "/pos";
    public static readonly string[] Capabilities = ["escpos", "scanner", "serial", "device.identity", "secure-store", "telemetry", "biometric"];
#else
    public const string FormFactor = "desktop";
    public const string StartRoute = "/apps";
    public static readonly string[] Capabilities = ["external.open", "print.system", "secure-store", "device.identity", "telemetry", "biometric"];
#endif
}
