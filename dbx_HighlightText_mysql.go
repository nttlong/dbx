package dbx

func mysql_create_dbx_HighlightText_function() string {
	ret := `CREATE FUNCTION dbx_HighlightText
(
    p_start_tag VARCHAR(50),
    p_end_tag VARCHAR(50),
    p_input_text TEXT,
    p_keywords TEXT
)
RETURNS TEXT CHARSET utf8mb4
DETERMINISTIC
BEGIN
    DECLARE v_result TEXT;
    DECLARE v_keyword VARCHAR(100);
    DECLARE v_pos INT DEFAULT 0;
    DECLARE v_next_space INT;
    DECLARE v_processed_keywords TEXT;

    SET v_result = p_input_text;

    -- Sửa lỗi ở đây: Thay '||' bằng CONCAT()
    SET v_processed_keywords = CONCAT(LTRIM(RTRIM(p_keywords)), ' ');

    WHILE LOCATE(' ', v_processed_keywords, v_pos + 1) > 0 DO
        SET v_next_space = LOCATE(' ', v_processed_keywords, v_pos + 1);
        SET v_keyword = LTRIM(RTRIM(SUBSTRING(v_processed_keywords, v_pos + 1, v_next_space - v_pos - 1)));

        IF CHAR_LENGTH(v_keyword) > 0 THEN
            SET v_result = REGEXP_REPLACE(
                v_result,
                v_keyword,
                CONCAT(p_start_tag, v_keyword, p_end_tag),
                1,
                0,
                'i'
            );
        END IF;

        SET v_pos = v_next_space;
    END WHILE;

    RETURN v_result;
END `
	return ret
}
