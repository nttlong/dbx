package dbx

func mysql_create_dbx_HighlightText_function() string {
	ret := `CREATE FUNCTION dbx_HighlightText
(
    p_start_tag VARCHAR(50),  -- Tham số cho thẻ mở (ví dụ: '<b>')
    p_end_tag VARCHAR(50),    -- Tham số cho thẻ đóng (ví dụ: '</b>')
    p_input_text TEXT,        -- Văn bản gốc cần highlight (tương đương NVARCHAR(MAX))
    p_keywords TEXT           -- Các từ khóa cần highlight, cách nhau bởi dấu cách (tương đương NVARCHAR(MAX))
)
RETURNS TEXT                  -- Kiểu trả về là TEXT (tương đương NVARCHAR(MAX))
DETERMINISTIC                 -- Hàm luôn trả về cùng một kết quả cho cùng một đầu vào
BEGIN
    DECLARE v_result TEXT;             -- Biến lưu trữ kết quả cuối cùng
    DECLARE v_keyword VARCHAR(100);    -- Biến lưu trữ từng từ khóa
    DECLARE v_pos INT DEFAULT 0;       -- Vị trí hiện tại trong chuỗi từ khóa
    DECLARE v_next_space INT;          -- Vị trí dấu cách tiếp theo
    DECLARE v_processed_keywords TEXT; -- Chuỗi từ khóa đã được xử lý để đơn giản hóa việc tách

    -- Khởi tạo biến kết quả bằng văn bản đầu vào
    SET v_result = p_input_text;

    -- Thêm dấu cách vào cuối chuỗi từ khóa để đơn giản hóa việc tách các từ
    -- MySQL sử dụng '||' hoặc CONCAT() để nối chuỗi
    SET v_processed_keywords = LTRIM(RTRIM(p_keywords)) || ' ';

    -- Vòng lặp để xử lý từng từ khóa
    -- LOCATE(substring, string, start_position) tương đương CHARINDEX trong SQL Server
    WHILE LOCATE(' ', v_processed_keywords, v_pos + 1) > 0 DO
        -- Tìm vị trí của dấu cách tiếp theo
        SET v_next_space = LOCATE(' ', v_processed_keywords, v_pos + 1);

        -- Trích xuất từ khóa hiện tại
        -- SUBSTRING(string, start, length) tương đương SUBSTRING trong SQL Server
        SET v_keyword = LTRIM(RTRIM(SUBSTRING(v_processed_keywords, v_pos + 1, v_next_space - v_pos - 1)));

        -- Chỉ thay thế nếu từ khóa không rỗng
        -- CHAR_LENGTH(string) tương đương LEN trong SQL Server (đếm ký tự)
        IF CHAR_LENGTH(v_keyword) > 0 THEN
            -- Thay thế từ khóa trong văn bản kết quả
            -- REGEXP_REPLACE(string, pattern, replacement, start_position, occurrence, match_type)
            -- 'i' là cờ cho tìm kiếm không phân biệt chữ hoa/thường (case-insensitive)
            -- 1: Bắt đầu tìm kiếm từ vị trí đầu tiên
            -- 0: Thay thế tất cả các lần xuất hiện
            SET v_result = REGEXP_REPLACE(
                v_result,
                v_keyword,
                CONCAT(p_start_tag, v_keyword, p_end_tag), -- Nối thẻ với từ khóa
                1,   -- Bắt đầu tìm kiếm từ vị trí 1
                0,   -- Thay thế tất cả các lần xuất hiện
                'i'  -- Cờ 'i' để tìm kiếm không phân biệt chữ hoa/thường
            );
        END IF;

        -- Di chuyển vị trí bắt đầu tìm kiếm cho lần lặp tiếp theo
        SET v_pos = v_next_space;
    END WHILE;

    -- Trả về văn bản đã được highlight
    RETURN v_result;
END`
	return ret
}
