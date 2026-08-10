//
//  AppDelegate.swift
//  GeregeNexusNativeMac
//
//  Created for Open Gerege Nexus Desktop Platform
//  Native Application Lifecycle & Menu Bar Integration
//

import Foundation
import AppKit

public class AppDelegate: NSObject, NSApplicationDelegate {
    public var mainWindowController: MainWindowController?
    private var settingsWindowController: SettingsWindowController?

    public func applicationDidFinishLaunching(_ notification: Notification) {
        setupNativeMenuBar()

        mainWindowController = MainWindowController()
        mainWindowController?.showWindow(nil)
        mainWindowController?.window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    public func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }

    private func setupNativeMenuBar() {
        let mainMenu = NSMenu()

        // 1. App Menu: Gerege Nexus
        let appMenuItem = NSMenuItem()
        let appMenu = NSMenu(title: "Gerege Nexus")
        appMenu.addItem(withTitle: "Gerege Nexus-ийн тухай", action: #selector(aboutApp), keyEquivalent: "")
        appMenu.addItem(NSMenuItem.separator())
        appMenu.addItem(withTitle: "Нэвтрэх цонх...", action: #selector(actionLogin), keyEquivalent: "l")
        appMenu.addItem(withTitle: "Тохиргоо...", action: #selector(actionPreferences), keyEquivalent: ",")
        appMenu.addItem(NSMenuItem.separator())
        appMenu.addItem(withTitle: "Gerege Nexus-ээс гарах", action: #selector(actionQuit), keyEquivalent: "q")
        appMenuItem.submenu = appMenu
        mainMenu.addItem(appMenuItem)

        // 2. Manage Menu: Удирдах
        let manageMenuItem = NSMenuItem()
        let manageMenu = NSMenu(title: "Удирдах")
        manageMenu.addItem(withTitle: "Апп дэлгүүр (/apps)", action: #selector(actionApps), keyEquivalent: "1")
        manageMenu.addItem(withTitle: "Дахин ачаалах", action: #selector(actionReload), keyEquivalent: "r")
        manageMenuItem.submenu = manageMenu
        mainMenu.addItem(manageMenuItem)

        // 3. View Menu: Харах
        let viewMenuItem = NSMenuItem()
        let viewMenu = NSMenu(title: "Харах")
        viewMenu.addItem(withTitle: "Бүтэн дэлгэцээр харах", action: #selector(toggleFullScreen), keyEquivalent: "f")
        viewMenuItem.submenu = viewMenu
        mainMenu.addItem(viewMenuItem)

        NSApp.mainMenu = mainMenu
    }

    @objc private func aboutApp() {
        let alert = NSAlert()
        alert.messageText = "Open Gerege Nexus Native"
        alert.informativeText = "Gerege Nexus Enterprise Native Shell v1.0\nBuilt on Swift 5.10 + AppKit + WKWebView"
        alert.alertStyle = .informational
        alert.addButton(withTitle: "Ойлголоо")
        alert.runModal()
    }

    @objc private func actionLogin() {
        mainWindowController?.showNativeLogin()
    }

    @objc private func actionPreferences() {
        if settingsWindowController == nil { settingsWindowController = SettingsWindowController() }
        settingsWindowController?.showWindow(nil)
        settingsWindowController?.window?.makeKeyAndOrderFront(nil)
    }

    @objc private func actionApps() {
        mainWindowController?.loadRelativePath("/apps")
    }

    @objc private func actionReload() {
        mainWindowController?.reloadPage()
    }

    @objc private func toggleFullScreen() {
        mainWindowController?.window?.toggleFullScreen(nil)
    }

    @objc private func actionQuit() {
        NSApp.terminate(nil)
    }
}
