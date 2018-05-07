package main

/**
*  用将YAML文件转化为结构体。
 */

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

//将一个字符串的首字母大写的方法
func strFirstToUpper(str string) string {
	//如果字符串为空 直接返回空
	if len(str) == 0 {
		return ""
	}
	//如果字符串只有一个字符 直接返回该字符的大写
	if len(str) == 1 {
		return strings.ToUpper(str)
	}
	//长度超过1 则先取首字母的大写
	upperHead := strings.ToUpper(string(str[0]))
	//返回首字母大写和其他字符的组合
	return upperHead + str[1:]
}

//名称驼峰化的方法
//将形如group_Id的名称格式化为GroupId、groupId 形如name的名称格式化为Name、name
func formatToHump(yamlName string) (structName, jsonName string) {
	//字符串为空则直接返回
	if len(yamlName) == 0 {
		return "", ""
	}
	//去掉字符串左边的下划线
	yamlName = strings.TrimLeft(yamlName, "_")
	//去掉字符串右边的下划线
	yamlName = strings.TrimRight(yamlName, "_")
	//将字符串以_切分
	splitedString := strings.Split(yamlName, "_") //将字符串以_来切分
	//遍历切分过的字符串数组 将除了第一个元素之外后面的全部首字母大写
	for i, v := range splitedString {
		if i == 0 {
			//第一个字符串 只有结构体的需要大写
			structName += strFirstToUpper(v)
			jsonName += v
		} else {
			//除了第一个元素之外的全部首字母大写
			structName += strFirstToUpper(v)
			jsonName += strFirstToUpper(v)
		}
	}
	return structName, jsonName
}

//根据给出的yaml文件路径 提取出文件名
func getYamlName(filePath string) (fileName, warnMsg string) {
	//文件路径为空 返回空
	if filePath == "" {
		return "", "请输入文件路径"
	} else if len(filePath) < 6 {
		//yaml文件应该形如a.yaml ab.yaml等等 路径至少应该6位
		return "", "请如入正确的文件路径"
	} else if filePath[len(filePath)-5:] != ".yaml" {
		//提取文件扩展名 规则是按.切分 最后一个元素是扩展名
		filePathSplit := strings.Split(filePath, ".")
		//yaml文件路径后五位应该是.yaml
		return "", "不支持的文件类型：" + filePathSplit[len(filePathSplit)-1]
	}
	//判断是否包含路径分隔符
	if strings.Contains(filePath, "/") {
		//如果包含linux路径 以linux路径分隔符进行切分
		filePathSplit := strings.Split(filePath, "/")
		//取最后一个元素
		containsFileName := filePathSplit[len(filePathSplit)-1]
		//判断最后一个元素长度是不是小于6位
		if len(containsFileName) < 6 {
			return "", "请输入文件名"
		}
		//不是路径的话 只是文件名 直接取从开头到倒数第6位 就是文件名
		return containsFileName[:len(containsFileName)-5], ""
	} else if strings.Contains(filePath, "\\") {
		//如果包含windows路径 以windows路径分隔符进行切分
		filePathSplit := strings.Split(filePath, "\\")
		//取最后一个元素
		containsFileName := filePathSplit[len(filePathSplit)-1]
		//判断最后一个元素长度是不是小于6位
		if len(containsFileName) < 6 {
			return "", "请输入文件名"
		}
		//不是路径的话 只是文件名 直接取从开头到倒数第6位 就是文件名
		return containsFileName[:len(containsFileName)-5], ""
	}
	//不是路径的直接取文件名
	return filePath[:len(filePath)-5], ""
}

//获取每一行冒号后面的内容中第一个逗号前面的部分
//如，某一行内容为：description:姓名，不可为空 本方法的作用就是获取姓名二字
func getValueAfterColon(lineStr string) (value, errMsg string) {
	//获取第一个冒号的位置
	colonIndex := strings.Index(lineStr, ":")
	//位置为-1 说明不含冒号
	if colonIndex == -1 {
		return "", "不含冒号，格式有问题"
	} else if colonIndex == (len(lineStr) - 1) {
		//冒号在最后一个位置 无value
		return "", "本行值为空"
	}
	//提取本行value部分 粗粒度提取
	lineValue := lineStr[colonIndex+1:]
	//对value部分进行细粒度提取 即提取逗号或者冒号前面的部分
	if strings.Contains(lineValue, ",") {
		//value包含逗号 获取第一个逗号的位置
		commaIndex := strings.Index(lineValue, ",")
		//如果逗号在第一个字符位置
		if commaIndex == 0 && len(lineValue) == 1 {
			//同时该字符串只有1个字符 还是不放过了吧...
			return "", `本行值是：","不符合业务需求`
		}
		return lineValue[:commaIndex], ""
	} else if strings.Contains(lineValue, ":") {
		//value包含冒号 获取第一个冒号的位置
		colonIndex := strings.Index(lineValue, ":")
		//如果冒号在第一个字符位置
		if colonIndex == 0 && len(lineValue) == 1 {
			//同时该字符串只有1个字符 还是不放过了吧...
			return "", `本行值是：":"不符合业务需求`
		}
		return lineValue[:colonIndex], ""
	}
	//不含冒号或者逗号的情况 直接返回第一个冒后面的
	return lineValue, ""
}

//以yaml文件路径为参数 将表信息转化为结构体
func convertToStruct(filePath string) (int, string, error) {
	/*-----------------1.读取yaml文件-----------------*/
	yamlFile, err := os.Open(filePath)
	if err != nil {
		//发生错误则返回
		return 0, "", err
	}
	//文件使用结束时关闭文件
	defer yamlFile.Close()
	/*-----------------2.创建或打开保存结果的文件-----------------*/
	//获取文件名
	fileName, warnMsg := getYamlName(filePath)
	//如果有错误提示 则返回提示信息
	if warnMsg != "" {
		return 0, "", errors.New(warnMsg)
	}
	//拼接存储结果的文件的文件名
	resultSaveFileName := fileName + ".txt"
	//创建或打开存储提取结果的文件 并以追加的形式写入
	resultSaveFile, err := os.OpenFile(resultSaveFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	//structFile, err := os.OpenFile("struct.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644) //清空原有内容
	if err != nil {
		//发生错误则返回
		return 0, "", err
	}
	//文件使用结束时关闭文件
	defer resultSaveFile.Close()
	/*-----------------3.逐行读取yaml文件 提取所需信息-----------------*/
	//分别用来存储表中文名、主键序列、表英文名、结构体信息、数据校验信息、
	//json信息、new对象时赋值信息以及给前端的对接文档信息
	var tableCHName, primaryKey, tableName, structInfo,
		dataChekInfo, jsonInfo, newObjInfo, docInfo string
	//读取每个字段时 存储该字段的相关信息 分别是属性类型、属性中文名、
	//机构体属性名、json属性名、属性最大长度，是否定长和是否可为空
	var columnType, columnChName, columnStructName, columnJsonName,
		columnLength, columnFixedLen, columnNullable string
	var lineCount int     //记录读取过的行数
	var attrCount int = 1 //记录当前是读取到的第几个属性
	var jsonValue string  //json字符串value部分的默认值 如果是数字则是0
	var indent = "\t"     //定义每一行的缩进
	var newLine = "\n"    //换行符
	//获取当前时间
	date := time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05")
	//初始化各项信息
	summary := newLine + "--------------------------------生成时间：" + //汇总信息
		date + "--------------------------------" + newLine //记录每次生成的时间
	tableCHName += "*表名" + newLine
	primaryKey += "*主键序列" + newLine
	structInfo += "*结构体定义" + newLine
	dataChekInfo += "*数据校验" + newLine
	//json字符串的开头部分
	jsonInfo += `*对应的json字符串` + newLine + indent + `{` + newLine +
		indent + indent + `"com":"",` + newLine +
		indent + indent + `"data":{"` + newLine
	newObjInfo += "*初始化对象时给字段赋值用" + newLine
	docInfo += "*给前端的接口文档用(入参以及出参字段说明)" + newLine
	//初始化一个文件读取对象
	scanner := bufio.NewScanner(yamlFile)
	//开始逐行读取文件内容
	for scanner.Scan() {
		//计数器自增
		lineCount++
		//获取该行文本内容
		line := scanner.Text()
		lineStr := strings.Replace(line, "\t", "", -1)   //去tab空格
		lineStr = strings.Replace(lineStr, " ", "", -1)  //去空格
		lineStr = strings.Replace(lineStr, "，", ",", -1) //逗号转换为半角的
		lineStr = strings.Replace(lineStr, "。", ".", -1) //句号转换为半角的
		lineStr = strings.Replace(lineStr, "：", ":", -1) //冒号转换为半角的
		//获取表名
		if strings.Contains(lineStr, "table-name:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//获取驼峰式的表名
			tableName, _ = formatToHump(value)
		} else if strings.Contains(lineStr, "descripion:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//不带-的description是表的描述 作为表中文名提取
			tableCHName += indent + value + newLine
			//追加结构体信息 表中文名作为结构体的注释信息
			structInfo += indent + "//" + value + newLine + indent + "type " +
				tableName + " struct {" + newLine
		} else if strings.Contains(lineStr, "-description:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//获取字段名称 保存变量中文名
			columnChName = value
			//判断是否定长
			if strings.Contains(lineStr, "定长") && !strings.Contains(lineStr, "不定长") {
				//暂时以是否含有定长两字作为标准
				columnFixedLen = "true"
			} else {
				//其他都是不定长
				columnFixedLen = "false"
			}
		} else if strings.Contains(lineStr, "type:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//转为小写
			valueLower := strings.ToLower(value)
			//获取字段类型
			if valueLower == "string" {
				//字符串类型 妥妥的string
				columnType = "string"
				jsonValue = `""` //json字符串value部分改为空字符串
			} else if valueLower == "number" && (strings.ContainsAny(columnChName, "价金额") ||
				strings.Contains(columnChName, "系数")) {
				//价格 系数等浮点类型 设置为float64
				columnType = "float64"
				jsonValue = `0` //json字符串value部分改为0
			} else if valueLower == "number" && !(strings.ContainsAny(columnChName, "价金额") ||
				strings.Contains(columnChName, "系数")) {
				//其他数字类型 设置为int
				columnType = "int"
				jsonValue = `0` //json字符串value部分改为0
			} else {
				//其他的设置为string
				columnType = "string"
				jsonValue = `""` //json字符串value部分 默认是空字符串
			}
		} else if strings.Contains(lineStr, "length:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//保存字段中文名
			columnLength = value
		} else if strings.Contains(lineStr, "nullable:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//转为小写
			valueLower := strings.ToLower(value)
			//保存该字段是否可为空的信息
			if valueLower == "yes" {
				columnNullable = "true"
			} else if valueLower == "no" {
				columnNullable = "false"
			} else {
				columnNullable = "false"
			}
		} else if strings.Contains(lineStr, "name:") && !strings.Contains(lineStr, "table-name:") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//获取结构体属性名和json属性名
			columnStructName, columnJsonName = formatToHump(value)
			//读到这一行 一个属性的所有信息就都读取完了 组织并添加各个信息
			//添加结构体信息
			structInfo += indent + indent + columnStructName + "\t" + columnType + "\t" +
				"`json:\"" + columnJsonName + "\"`" + "\t" + "//" + columnChName + newLine
			//数据校验信息 待校验的结构体对象名称定义为structNeededToCheck 方便替换
			dataChekInfo += indent + `util.CheckParam(structNeededToCheck.` + columnStructName +
				`, "` + columnChName + `", ` + columnLength + `, ` + columnFixedLen + `, ` +
				columnNullable + `)` + newLine
			//添加json信息
			if attrCount == 1 {
				//如果是第一个属性就直接添加
				jsonInfo += indent + indent + indent + `"` + columnJsonName + `":` + jsonValue
			} else {
				//如果不是第一个就在前面加一个逗号和换行符 补全上一行
				jsonInfo += "," + newLine + indent + indent + indent + `"` +
					columnJsonName + `":` + jsonValue
			}
			//添加new对象时赋值信息 dataSource是该字段值来源 方便查找替换
			newObjInfo += indent + columnStructName + ": dataSource." + columnStructName + ", //" +
				columnChName + newLine
			//添加给前端的对接文档信息
			docInfo += indent + columnJsonName + "\t\t" + columnChName + newLine
			//记录的属性数加1
			attrCount++
		} else if strings.Contains(lineStr, "primary-keys") {
			//获取本行的value部分
			value, errMsg := getValueAfterColon(lineStr)
			//有错误提示 返回提示
			if errMsg != "" {
				//组织错误提示信息
				errMsg = "第" + fmt.Sprintf("%d", lineCount) + "行，内容是：" + line + "，" + errMsg
				return 0, "", errors.New(errMsg)
			}
			//读到含有primary-keys的行 说明该所列出的字段已经全部读取 可以结束了
			//结束主键序列信息
			primaryKey += indent + value + newLine
			//结束结构体信息
			structInfo += indent + "}" + newLine
			//结束json信息
			jsonInfo += newLine + indent + indent + `}` + newLine + indent + `}` + newLine
			//将各个信息整合在一起
			date = time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05") //获取最新时间
			summary += newLine + tableCHName + newLine + primaryKey + newLine +
				structInfo + newLine + jsonInfo + newLine + dataChekInfo + newLine +
				newObjInfo + newLine + docInfo + newLine +
				"================================信息提取结束 " + //提取结束的信息
				date + "================================" + newLine //记录每次生成的时间
			//结束读取
			break
		}
	}
	//开始将整合后的信息写入到文件当中
	writeLineCount, err := resultSaveFile.WriteString(summary)
	//出错的话 返回错误
	if err != nil {
		return 0, "", err
	}
	//无错返回nil
	return writeLineCount, resultSaveFileName, nil
}

func main() {
	//从命令行参数获取文件路径
	args := os.Args
	if args == nil || len(args) < 2 {
		//如果用户没有输入文件路径则提示用户
		log.Println("请输入文件路径")
		return
	}
	//获取文件路径
	filePath := args[1]
	//读取该路径的信息
	filePathInfo, err := os.Stat(filePath)
	if err != nil {
		//异常处理
		log.Println("读取路径信息异常，错误信息：", err)
		return
	}
	//判断是否是目录 然后再进一步做处理
	if filePathInfo.IsDir() {
		//是目录的话 读取目录下的文件 不包含子目录
		dirList, e := ioutil.ReadDir(filePath)
		if e != nil {
			log.Println("读取目录：%d下的文件列表异常，错误信息：%v\n", filePath, e)
			return
		}
		//转换状况的汇总信息
		var fileAmt, sucAmt, failAmt int //总转换数量、成功的数量、失败数量
		var sucMsg, failMsg string       //成功提示信息、失败提示信息
		//对filePath进行处理 没有/则添加/
		if !strings.HasSuffix(filePath, "/") {
			filePath += "/"
		}
		//循环遍历该文件夹下的文件 对yaml进行处理
		for _, dirInfo := range dirList {
			if !dirInfo.IsDir() && strings.HasSuffix(dirInfo.Name(), ".yaml") {
				//不是目录的话且是yaml文件的话才转换 拼接该文件的路径
				fileDir := filePath + dirInfo.Name()
				//总转换数量加1
				fileAmt++
				//开始提取yaml内的信息
				_, _, err := convertToStruct(fileDir)
				if err != nil {
					//失败数量加1
					failAmt++
					//异常处理
					failMsg += "\t" + fmt.Sprintf("%d", failAmt) + "." + "文件：" +
						dirInfo.Name() + "转换失败，失败信息：" + err.Error() + "\n"
				} else {
					//成功的数量加1
					sucAmt++
				}
			}
		}
		//遍历完组织转换详情返回提示
		if failAmt == 0 {
			//没有转换失败的
			sucMsg = "全部转换成功！"
		} else {
			//有转换失败的
			sucMsg = "其中" + fmt.Sprintf("%d", sucAmt) + "个转换成功，"
			failMsg = "，" + fmt.Sprintf("%d", failAmt) + "个转换失败：\n" + failMsg
		}
		//转换结束输出提示信息
		log.Printf("共转换%d个文件，%v，转换结果已保存到目录：%v下%v\n",
			fileAmt, sucMsg, filePath, failMsg)
	} else {
		//是文件的话 看是否是yaml文件
		if strings.HasSuffix(filePathInfo.Name(), ".yaml") {
			//开始提取yaml内的信息 第一个返回值是写入的字符串长度(中文3英文1)
			_, resultFileName, err := convertToStruct(filePath)
			if err != nil {
				//异常处理
				log.Printf("文件路径是：%v，提取yaml文件信息异常，错误信息是：%v\n", filePath, err)
				return
			}
			//无错提示成功
			log.Printf("文件：%v，提取yaml文件成功，已将信息提取到%v文件中\n",
				filePathInfo.Name(), resultFileName)
		} else {
			//不是yaml文件 给出提示
			log.Printf("文件：%v不是yaml文件，请选择路径并重新运行本程序\n", filePathInfo.Name())
		}
	}
}
