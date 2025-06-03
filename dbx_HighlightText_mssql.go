package dbx

func createMssqlHighlightFunction() string {
	ret := `CREATE OR ALTER FUNCTION [dbo].[dbx_HighlightText]
		(
			@StartTag NVARCHAR(50),
			@EndTag NVARCHAR(50),
			@InputText NVARCHAR(MAX),
			@Keywords NVARCHAR(MAX)
		)
		RETURNS NVARCHAR(MAX)
		AS
		BEGIN
			DECLARE @Result NVARCHAR(MAX)
			SET @Result = @InputText

			DECLARE @Keyword NVARCHAR(100)
			DECLARE @Pos INT = 0

			-- Thêm dấu cách vào đầu và cuối để đơn giản hóa tìm kiếm
			SET @Keywords = LTRIM(RTRIM(@Keywords)) + ' '

			WHILE CHARINDEX(' ', @Keywords, @Pos + 1) > 0
			BEGIN
				DECLARE @NextSpace INT = CHARINDEX(' ', @Keywords, @Pos + 1)
				SET @Keyword = LTRIM(RTRIM(SUBSTRING(@Keywords, @Pos + 1, @NextSpace - @Pos - 1)))

				-- Chỉ thay thế nếu keyword tồn tại và không trống
				IF LEN(@Keyword) > 0
				BEGIN
					-- Thay thế từ khóa (không phân biệt hoa thường)
					SET @Result = REPLACE(
						@Result,
						@Keyword,
						@StartTag + @Keyword + @EndTag
					)
				END

				SET @Pos = @NextSpace
			END

			RETURN @Result
		END
		`
	return ret
}
