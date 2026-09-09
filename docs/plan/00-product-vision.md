# InfoBoard: Tầm nhìn sản phẩm

## Định nghĩa một câu

**InfoBoard là dashboard thông tin dạng text, local-first, giúp lưu lại, tổ chức và tìm lại thông tin cá nhân.**

Ứng dụng không chủ yếu là trình đọc tài liệu hay ứng dụng ghi chú tổng quát. Nhiệm vụ cốt lõi là biến text rải rác thành một dashboard trực quan, có thể tìm kiếm và thấy được các mối liên hệ.

## Vấn đề cần giải quyết

Thông tin hữu ích thường nằm rải rác trong bài viết web, PDF, Markdown, text được sao chép và ghi chú cá nhân. Công cụ bookmark truyền thống chỉ giữ URL nên khó tìm lại; thư mục thông thường chỉ giữ file nhưng không làm lộ ra mối quan hệ giữa nội dung bên trong.

InfoBoard lưu bản snapshot local của nội dung, đặt nội dung vào collections/projects do người dùng định nghĩa, và cho phép tìm lại bằng từ khóa chính xác hoặc theo ngữ nghĩa.

## Người dùng mục tiêu và nguyên tắc

Phiên bản đầu phục vụ một người dùng trên chính máy của họ.

- **Local-first:** không cần tài khoản, đồng bộ hay workspace chia sẻ.
- **Text-first:** text trích xuất là nội dung chính; ảnh web ngoài phạm vi MVP.
- **Tổ chức theo project:** một item có thể liên kết với nhiều collection.
- **Dashboard hữu ích:** analytics phải giúp người dùng thấy mình đã lưu, đã đọc và đang tích lũy gì.
- **Riêng tư mặc định:** embedding local là mặc định; embedding cloud là lựa chọn chủ động.

## Đối tượng thông tin cốt lõi

Một **item** là một mẩu thông tin đã lưu. Nó có thể đến từ bài viết web công khai, PDF, Markdown, file text hoặc text nhập tay. Mỗi item có tiêu đề, snapshot nội dung, metadata nguồn, trạng thái đọc, collections, một note cá nhân tùy chọn và trạng thái index.

## Kết quả MVP

Người dùng có thể lưu bài viết hoặc file, đọc lại snapshot text local, tổ chức nội dung vào nhiều project, ghi note, tìm bằng từ khóa hoặc ngữ nghĩa, và xem dashboard tổng quan về thư viện của mình.

