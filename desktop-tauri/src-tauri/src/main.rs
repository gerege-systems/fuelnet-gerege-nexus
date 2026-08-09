// Windows дээр release build нь консолын цонх нээхгүй байх ёстой.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    gerege_nexus_shell_lib::run()
}
