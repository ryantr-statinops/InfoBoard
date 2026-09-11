# Design — Information architecture

## Navigation model

```text
InfoBoard
├── Dashboard
├── All items
├── Collections
├── Inbox
├── Active
├── Archived
└── Settings / advanced maintenance
```

## Dashboard structure

```text
┌──────────────┬──────────────────────────────────────────────┐
│ Navigation   │ Search                         + Thêm       │
│              ├──────────────────────────────────────────────┤
│              │ Collection · Nguồn · Trạng thái · Thời gian │
│              ├──────────────────────────────────────────────┤
│              │ KPI và trạng thái xử lý                     │
│              ├────────────────────────┬─────────────────────┤
│              │ Danh sách gần đây      │ Analytics/insights  │
└──────────────┴────────────────────────┴─────────────────────┘
                         Item → detail panel
```

## URL state

Filter/search context gồm `q`, collection, source, status và date range. Mở/đóng detail không làm mất context; list mặc định loại archived và dùng khoảng 30 ngày, có lựa chọn toàn thời gian.

## Information rules

- Một item có nhiều collection nên tổng theo collection có thể lớn hơn tổng item.
- Job count toàn ứng dụng phải được ghi rõ khi filter khác chỉ áp dụng cho items/analytics.
- Advanced maintenance không chen vào flow capture/retrieve mặc định.
