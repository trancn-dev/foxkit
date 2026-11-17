use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::sync::Mutex;

pub const THREAD_MAX: usize = 100;

/// Một mục của luồng chat dùng chung (chỉ trong bộ nhớ, không ghi đĩa).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct ThreadItem {
    pub id: u32,
    pub line: String,
    pub status: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub result: Option<Value>,
    pub ts: u64,
}

#[derive(Default)]
pub struct Thread {
    inner: Mutex<(u32, Vec<ThreadItem>)>,
}

impl Thread {
    pub fn list(&self) -> Vec<ThreadItem> {
        self.inner.lock().unwrap().1.clone()
    }

    /// Gán id mới, giữ tối đa THREAD_MAX mục (bỏ cái cũ nhất).
    pub fn add(&self, mut item: ThreadItem) -> u32 {
        let mut g = self.inner.lock().unwrap();
        g.0 += 1;
        item.id = g.0;
        g.1.push(item);
        if g.1.len() > THREAD_MAX {
            let extra = g.1.len() - THREAD_MAX;
            g.1.drain(..extra);
        }
        g.0
    }

    pub fn update(&self, id: u32, mut item: ThreadItem) {
        let mut g = self.inner.lock().unwrap();
        if let Some(slot) = g.1.iter_mut().find(|x| x.id == id) {
            item.id = id;
            *slot = item;
        }
    }

    /// Xóa đúng một mục theo id; id không tồn tại thì bỏ qua.
    pub fn remove(&self, id: u32) {
        self.inner.lock().unwrap().1.retain(|x| x.id != id);
    }

    pub fn clear(&self) {
        self.inner.lock().unwrap().1.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn item(line: &str) -> ThreadItem {
        ThreadItem { id: 0, line: line.into(), status: "running".into(), result: None, ts: 0 }
    }

    #[test]
    fn add_assigns_increasing_ids_and_trims_oldest() {
        let t = Thread::default();
        for i in 0..105 {
            t.add(item(&format!("c{i}")));
        }
        let l = t.list();
        assert_eq!(l.len(), THREAD_MAX);
        assert_eq!(l[0].line, "c5");
        assert_eq!(l.last().unwrap().id, 105);
    }

    #[test]
    fn running_becomes_result_on_same_item() {
        let t = Thread::default();
        let id = t.add(item("uuid"));
        t.update(id, ThreadItem { status: "done".into(), result: Some(Value::Bool(true)), ..item("uuid") });
        let l = t.list();
        assert_eq!((l[0].id, l[0].status.as_str()), (id, "done"));
        t.clear();
        assert!(t.list().is_empty());
    }

    #[test]
    fn remove_deletes_only_that_item() {
        let t = Thread::default();
        let a = t.add(item("a"));
        let b = t.add(item("b"));
        t.remove(a);
        t.remove(999);
        let l = t.list();
        assert_eq!(l.len(), 1);
        assert_eq!(l[0].id, b);
    }
}
