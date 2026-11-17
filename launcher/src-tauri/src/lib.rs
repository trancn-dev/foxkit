pub fn run() {
    tauri::Builder::default()
        .run(tauri::generate_context!())
        .expect("lỗi khi chạy foxkit launcher");
}
