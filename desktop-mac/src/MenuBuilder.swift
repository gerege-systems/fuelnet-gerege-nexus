import Cocoa

class MenuBuilder {
    
    static func setupMainMenu(appDelegate: AppDelegate) -> NSMenu {
        let mainMenu = NSMenu()
        
        // 1. App Menu
        let appMenuItem = NSMenuItem()
        mainMenu.addItem(appMenuItem)
        let appMenu = NSMenu()
        appMenuItem.submenu = appMenu
        
        appMenu.addItem(withTitle: "Open Gerege Nexus-ийн тухай", action: #selector(appDelegate.showAbout), keyEquivalent: "")
        appMenu.addItem(withTitle: "Шинэчлэлт шалгах...", action: #selector(appDelegate.checkUpdates), keyEquivalent: "")
        appMenu.addItem(NSMenuItem.separator())
        
        let prefsItem = NSMenuItem(title: "Тохиргоо...", action: #selector(appDelegate.openPreferences), keyEquivalent: ",")
        appMenu.addItem(prefsItem)
        appMenu.addItem(NSMenuItem.separator())
        
        let servicesItem = NSMenuItem(title: "Сервисүүд", action: nil, keyEquivalent: "")
        let servicesMenu = NSMenu()
        servicesItem.submenu = servicesMenu
        NSApplication.shared.servicesMenu = servicesMenu
        appMenu.addItem(servicesItem)
        appMenu.addItem(NSMenuItem.separator())
        
        appMenu.addItem(withTitle: "Gerege Nexus-ийг нуух", action: #selector(NSApplication.hide(_:)), keyEquivalent: "h")
        let hideOthers = NSMenuItem(title: "Бусдыг нуух", action: #selector(NSApplication.hideOtherApplications(_:)), keyEquivalent: "h")
        hideOthers.keyEquivalentModifierMask = [.command, .option]
        appMenu.addItem(hideOthers)
        appMenu.addItem(withTitle: "Бүгдийг харуулах", action: #selector(NSApplication.unhideAllApplications(_:)), keyEquivalent: "")
        appMenu.addItem(NSMenuItem.separator())
        appMenu.addItem(withTitle: "Гарах", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        
        // 2. File Menu
        let fileMenuItem = NSMenuItem()
        mainMenu.addItem(fileMenuItem)
        let fileMenu = NSMenu(title: "Файл")
        fileMenuItem.submenu = fileMenu
        
        fileMenu.addItem(withTitle: "Шинэ баримт...", action: #selector(appDelegate.newDocument), keyEquivalent: "n")
        fileMenu.addItem(withTitle: "PDF нээж гарын үсэг зурах...", action: #selector(appDelegate.openPDFForSigning), keyEquivalent: "o")
        fileMenu.addItem(NSMenuItem.separator())
        fileMenu.addItem(withTitle: "Хэвлэх / PDF болгох...", action: #selector(appDelegate.printDocument), keyEquivalent: "p")
        fileMenu.addItem(NSMenuItem.separator())
        fileMenu.addItem(withTitle: "Цонх хаах", action: #selector(NSWindow.performClose(_:)), keyEquivalent: "w")
        
        // 3. Edit Menu
        let editMenuItem = NSMenuItem()
        mainMenu.addItem(editMenuItem)
        let editMenu = NSMenu(title: "Засвар")
        editMenuItem.submenu = editMenu
        
        editMenu.addItem(withTitle: "Буцаах (Undo)", action: #selector(UndoManager.undo), keyEquivalent: "z")
        let redoItem = NSMenuItem(title: "Дахин хийх (Redo)", action: #selector(UndoManager.redo), keyEquivalent: "Z")
        redoItem.keyEquivalentModifierMask = [.command, .shift]
        editMenu.addItem(redoItem)
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(withTitle: "Огтлох (Cut)", action: #selector(NSTextView.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: "Хуулах (Copy)", action: #selector(NSTextView.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: "Буулгах (Paste)", action: #selector(NSTextView.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: "Бүгдийг сонгох", action: #selector(NSTextView.selectAll(_:)), keyEquivalent: "a")
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(withTitle: "Хайх...", action: #selector(appDelegate.focusSearch), keyEquivalent: "f")
        
        // 4. View Menu
        let viewMenuItem = NSMenuItem()
        mainMenu.addItem(viewMenuItem)
        let viewMenu = NSMenu(title: "Харагдац")
        viewMenuItem.submenu = viewMenu
        
        viewMenu.addItem(withTitle: "Дахин ачаалах", action: #selector(appDelegate.reloadPage), keyEquivalent: "r")
        let forceReload = NSMenuItem(title: "Хүчээр ачаалах", action: #selector(appDelegate.forceReloadPage), keyEquivalent: "R")
        forceReload.keyEquivalentModifierMask = [.command, .shift]
        viewMenu.addItem(forceReload)
        viewMenu.addItem(NSMenuItem.separator())
        let fullScreen = NSMenuItem(title: "Бүтэн дэлгэц", action: #selector(NSWindow.toggleFullScreen(_:)), keyEquivalent: "f")
        fullScreen.keyEquivalentModifierMask = [.command, .control]
        viewMenu.addItem(fullScreen)
        viewMenu.addItem(NSMenuItem.separator())
        
        // Zoom
        viewMenu.addItem(withTitle: "Томсгох", action: #selector(appDelegate.zoomIn), keyEquivalent: "+")
        viewMenu.addItem(withTitle: "Жижигсгэх", action: #selector(appDelegate.zoomOut), keyEquivalent: "-")
        viewMenu.addItem(withTitle: "Хэвийн хэмжээ", action: #selector(appDelegate.zoomReset), keyEquivalent: "0")
        viewMenu.addItem(NSMenuItem.separator())
        let devTools = NSMenuItem(title: "Хөгжүүлэгчийн хэрэгсэл", action: #selector(appDelegate.toggleDevTools), keyEquivalent: "i")
        devTools.keyEquivalentModifierMask = [.command, .option]
        viewMenu.addItem(devTools)
        
        // 5. Apps Menu
        let appsMenuItem = NSMenuItem()
        mainMenu.addItem(appsMenuItem)
        let appsMenu = NSMenu(title: "Аппликейшнүүд")
        appsMenuItem.submenu = appsMenu
        
        let appList = [
            ("Апп Стор (Платформ)", "/apps", "0"),
            ("1. Харилцагчид (Contacts)", "/contacts", "1"),
            ("2. Бараа бүтээгдэхүүн (Products)", "/products", "2"),
            ("3. Агуулах (Inventory)", "/inventory", "3"),
            ("4. Нэхэмжлэх ба e-Barimt", "/billing", "4"),
            ("5. Цахим баримт бичиг", "/documents", "5"),
            ("6. PDF Цахим гарын үсэг", "/esign", "6"),
            ("7. Хөгжүүлэгчийн портал", "/developer/apps", "7"),
            ("8. Төрийн үйлчилгээ", "/gov-services", "8"),
        ]
        
        for item in appList {
            let menuItem = NSMenuItem(title: item.0, action: #selector(appDelegate.navigateToApp(_:)), keyEquivalent: item.2)
            menuItem.representedObject = item.1
            appsMenu.addItem(menuItem)
        }
        
        // 6. Window Menu
        let windowMenuItem = NSMenuItem()
        mainMenu.addItem(windowMenuItem)
        let windowMenu = NSMenu(title: "Цонх")
        windowMenuItem.submenu = windowMenu
        NSApplication.shared.windowsMenu = windowMenu
        
        windowMenu.addItem(withTitle: "Жижигсгэх", action: #selector(NSWindow.performMiniaturize(_:)), keyEquivalent: "m")
        windowMenu.addItem(withTitle: "Томсгох", action: #selector(NSWindow.performZoom(_:)), keyEquivalent: "")
        windowMenu.addItem(NSMenuItem.separator())
        windowMenu.addItem(withTitle: "Бүх цонхыг урд гаргах", action: #selector(NSApplication.arrangeInFront(_:)), keyEquivalent: "")
        
        // 7. Help Menu
        let helpMenuItem = NSMenuItem()
        mainMenu.addItem(helpMenuItem)
        let helpMenu = NSMenu(title: "Тусламж")
        helpMenuItem.submenu = helpMenu
        NSApplication.shared.helpMenu = helpMenu
        
        helpMenu.addItem(withTitle: "Open Gerege Nexus Баримт бичиг", action: #selector(appDelegate.openDocumentation), keyEquivalent: "")
        helpMenu.addItem(withTitle: "Gerege Systems Дэмжлэг", action: #selector(appDelegate.openSupport), keyEquivalent: "")
        
        return mainMenu
    }
}
