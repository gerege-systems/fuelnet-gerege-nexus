using System;
using System.Windows;
using Microsoft.Web.WebView2.Core;
using QRCoder;
using System.IO;
using System.Windows.Media.Imaging;

namespace GeregeNexusNativeWin
{
    public partial class MainWindow : Window
    {
        private readonly NativeSettings _settings = NativeSettings.Load();
        private readonly string _baseUrl;
        private NativeIPCBridge? _ipcBridge;
        private readonly NativeAuth _auth;
        public string WebOrigin => new Uri(_baseUrl).GetLeftPart(UriPartial.Authority);

        public MainWindow()
        {
            _baseUrl = _settings.WebEndpoint.TrimEnd('/');
            _auth = new NativeAuth(_settings.ApiEndpoint);
            InitializeComponent();
            footerPlatform.Text = $"Windows · {ShellProfile.FormFactor}  |  Shell v1.3";
            emailInput.ToolTip=NativeStrings.Get("auth_field_email","И-мэйл");passwordInput.ToolTip=NativeStrings.Get("auth_field_password","Нууц үг");passwordLoginButton.Content=NativeStrings.Get("auth_action_admin_sign_in","Админаар нэвтрэх");nationalIdInput.ToolTip=NativeStrings.Get("auth_eid_reg_number","Регистрийн дугаар");pushLoginButton.Content=NativeStrings.Get("auth_eid_send_request","eID апп руу хүсэлт илгээх");cancelLoginButton.Content=NativeStrings.Get("auth_action_cancel","Цуцлах");staffPinButton.Content=NativeStrings.Get("auth_action_staff_sign_in","Ээлжийн ажилтнаар нэвтрэх");
            staffPinPanel.Visibility = ShellProfile.FormFactor == "pos" ? Visibility.Visible : Visibility.Collapsed;
            _auth.StatusChanged += status => Dispatcher.Invoke(() => RenderLogin(status));
            InitializeWebViewAsync();
        }

        private async void InitializeWebViewAsync()
        {
            try
            {
                await webView.EnsureCoreWebView2Async(null);

                // Initialize Native C# IPC Bridge
                _ipcBridge = new NativeIPCBridge(webView, this);

                // Inject Native Bridge JS initializer into every document created
                await webView.CoreWebView2.AddScriptToExecuteOnDocumentCreatedAsync(@"
                  (() => {
                    if (window.GeregeShell) return;
                    const pending=new Map(),listeners=new Map(); let sequence=0;
                    window.__geregeShellResolve=(id,ok,value)=>{const p=pending.get(id);if(!p)return;pending.delete(id);ok?p.resolve(value):p.reject(new Error(String(value)))};
                    window.__geregeShellEmit=(name,payload)=>(listeners.get(name)||[]).slice().forEach(fn=>fn(payload));
                    window.GeregeShell=Object.freeze({version:'1.3',platform:'windows',formFactor:'" + ShellProfile.FormFactor + @"',capabilities:Object.freeze(" + System.Text.Json.JsonSerializer.Serialize(ShellProfile.Capabilities) + @"),
                      invoke(method,params={}){return new Promise((resolve,reject)=>{const id=String(++sequence);pending.set(id,{resolve,reject});window.chrome.webview.postMessage(JSON.stringify({id,method,params}))})},
                      on(name,handler){const list=listeners.get(name)||[];list.push(handler);listeners.set(name,list);return()=>{const i=list.indexOf(handler);if(i>=0)list.splice(i,1)}}});
                    document.documentElement.setAttribute('data-shell','windows');
                  })();");

                // Listen to IPC messages sent from WebView2
                webView.CoreWebView2.WebMessageReceived += (s, e) =>
                {
                    string message = e.TryGetWebMessageAsString();
                    if (!string.IsNullOrEmpty(message))
                    {
                        _ipcBridge.HandleWebMessage(message, e.Source);
                    }
                };

                webView.CoreWebView2.NavigationStarting += NavigationStarting;
                webView.CoreWebView2.NavigationCompleted += (_, e) => { footerConnection.Text = e.IsSuccess ? $"●  Холбогдсон · {new Uri(webView.Source.ToString()).Host}" : "●  Холболт тасарсан"; footerConnection.Foreground = new System.Windows.Media.SolidColorBrush((System.Windows.Media.Color)System.Windows.Media.ColorConverter.ConvertFromString(e.IsSuccess ? "#62D9D4" : "#EF4444")); _ = webView.CoreWebView2.ExecuteScriptAsync("window.__geregeShellEmit&&window.__geregeShellEmit('shell:auth-changed',{reason:'navigation-session'})"); };
                var deviceToken = CredentialManagerTokenStore.Load();
                if (!string.IsNullOrWhiteSpace(deviceToken))
                {
                    var api = new Uri(_settings.ApiEndpoint);
                    var cookie = webView.CoreWebView2.CookieManager.CreateCookie("device_token", deviceToken, api.Host, "/api/v1/devices");
                    cookie.IsHttpOnly = true; cookie.IsSecure = api.Scheme == "https"; webView.CoreWebView2.CookieManager.AddOrUpdateCookie(cookie);
                    _ = new DeviceEnrollmentClient().TelemetryAsync(_settings.ApiEndpoint,deviceToken,"INFO","shell.started",new{form_factor=ShellProfile.FormFactor,runtime="webview2"});
                }
                if (ShellProfile.FormFactor == "kiosk" && !string.IsNullOrWhiteSpace(deviceToken)) { loginView.Visibility = Visibility.Collapsed; webView.Visibility = Visibility.Visible; nativeFooter.Visibility = Visibility.Visible; NavigatePath(ShellProfile.StartRoute); EnterKioskMode(); }
                else ShowNativeLogin();
            }
            catch (Exception ex)
            {
                MessageBox.Show($"WebView2 Initialization Error: {ex.Message}", "Gerege Nexus Native", MessageBoxButton.OK, MessageBoxImage.Error);
            }
        }

        public void NavigatePath(string relativePath)
        {
            if (webView != null && webView.CoreWebView2 != null)
            {
                string targetUrl = _baseUrl + relativePath;
                webView.CoreWebView2.Navigate(targetUrl);
            }
        }

        public void ShowNativeLogin()
        {
            loginView.Visibility = Visibility.Visible;
            webView.Visibility = Visibility.Collapsed;
            nativeFooter.Visibility = Visibility.Collapsed;
        }

        private async void PasswordLogin_Click(object sender, RoutedEventArgs e) =>
            await _auth.PasswordAsync(emailInput.Text, passwordInput.Password);

        private async void PushLogin_Click(object sender, RoutedEventArgs e) =>
            await _auth.PushAsync(nationalIdInput.Text);

        private async void QrLogin_Click(object sender, RoutedEventArgs e) => await _auth.QrAsync();

        private async void StaffPin_Click(object sender, RoutedEventArgs e)
        {
            var token = CredentialManagerTokenStore.Load();
            if (string.IsNullOrWhiteSpace(token)) { loginStatus.Text = "Эхлээд төхөөрөмжийг enrollment code-оор бүртгэнэ үү"; return; }
            await _auth.StaffPinAsync(staffPinInput.Password, token);
        }

        private void CancelLogin_Click(object sender, RoutedEventArgs e) => _auth.Cancel();

        private void RenderLogin(LoginStatus status)
        {
            var pending = status.Phase is LoginPhase.Starting or LoginPhase.Waiting;
            loginStatus.Text = status.Message;
            passwordLoginButton.IsEnabled = !pending; pushLoginButton.IsEnabled = !pending;
            qrLoginButton.IsEnabled = !pending;
            if (pending && !string.IsNullOrWhiteSpace(_auth.LastDeviceLinkUrl)) { using var data = QRCodeGenerator.GenerateQrCode(_auth.LastDeviceLinkUrl, QRCodeGenerator.ECCLevel.Q); var bytes = new PngByteQRCode(data).GetGraphic(8); var bitmap = new BitmapImage(); using var stream = new MemoryStream(bytes); bitmap.BeginInit(); bitmap.CacheOption = BitmapCacheOption.OnLoad; bitmap.StreamSource = stream; bitmap.EndInit(); bitmap.Freeze(); qrImage.Source = bitmap; qrImage.Visibility = Visibility.Visible; }
            else if (!pending) qrImage.Visibility = Visibility.Collapsed;
            cancelLoginButton.Visibility = pending ? Visibility.Visible : Visibility.Collapsed;
            if (status.Phase != LoginPhase.Success || webView.CoreWebView2 == null) return;
            foreach (System.Net.Cookie source in _auth.SessionCookies)
            {
                var cookie = webView.CoreWebView2.CookieManager.CreateCookie(source.Name, source.Value, source.Domain, source.Path);
                cookie.IsHttpOnly = source.HttpOnly; cookie.IsSecure = source.Secure;
                if (source.Expires != DateTime.MinValue) cookie.Expires = source.Expires;
                webView.CoreWebView2.CookieManager.AddOrUpdateCookie(cookie);
            }
            loginView.Visibility = Visibility.Collapsed; webView.Visibility = Visibility.Visible; nativeFooter.Visibility = Visibility.Visible;
            NavigatePath(ShellProfile.StartRoute);
        }

        private void NavigationStarting(object? sender, CoreWebView2NavigationStartingEventArgs e)
        {
            if (Uri.TryCreate(e.Uri, UriKind.Absolute, out var uri) && uri.GetLeftPart(UriPartial.Authority) == _baseUrl) return;
            e.Cancel = true;
            if (uri?.Scheme is "http" or "https" or "mailto" or "tel")
                System.Diagnostics.Process.Start(new System.Diagnostics.ProcessStartInfo(uri.AbsoluteUri) { UseShellExecute = true });
        }

        private void MenuAbout_Click(object sender, RoutedEventArgs e)
        {
            MessageBox.Show(
                "Open Gerege Nexus Native (Windows)\n\nGerege Nexus Enterprise Native Shell v1.0\nBuilt on C# .NET 8 + WPF + WebView2",
                "Gerege Nexus Native",
                MessageBoxButton.OK,
                MessageBoxImage.Information
            );
        }

        private void MenuLogin_Click(object sender, RoutedEventArgs e)
        {
            ShowNativeLogin();
        }

        private void MenuSettings_Click(object sender, RoutedEventArgs e)
        {
            new SettingsWindow(this).ShowDialog();
        }

        private void MenuApps_Click(object sender, RoutedEventArgs e)
        {
            NavigatePath("/apps");
        }

        private void MenuReload_Click(object sender, RoutedEventArgs e)
        {
            webView?.Reload();
        }

        private void MenuFullScreen_Click(object sender, RoutedEventArgs e)
        {
            if (WindowState == WindowState.Maximized)
            {
                WindowState = WindowState.Normal;
                WindowStyle = WindowStyle.SingleBorderWindow;
            }
            else
            {
                WindowStyle = WindowStyle.None;
                WindowState = WindowState.Maximized;
            }
        }

        private void EnterKioskMode() { WindowStyle = WindowStyle.None; WindowState = WindowState.Maximized; Topmost = true; }
        public void SetKioskMode(bool enabled) { if(enabled) EnterKioskMode(); else { Topmost=false;WindowState=WindowState.Normal;WindowStyle=WindowStyle.SingleBorderWindow; } }

        private void MenuExit_Click(object sender, RoutedEventArgs e)
        {
            Application.Current.Shutdown();
        }
    }
}
