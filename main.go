package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
	Status      string `json:"status"`
}

var db *sql.DB

func main() {
	// Open the SQLite database
	var err error
	db, err = sql.Open("sqlite3", "./task.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create the MANAGEMENT table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS MANAGEMENT (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			due_date DATE,
			status TEXT
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Setup and start the Gin web server
	router := gin.Default()

	// Define API endpoints
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, This is Task Management Api"})
	})

	router.POST("/MANAGEMENT", createTask)
	router.GET("/MANAGEMENT/:id", retrieveTask)
	router.PUT("/MANAGEMENT/:id", updateTask)
	router.DELETE("/MANAGEMENT/:id", deleteTask)
	router.GET("/MANAGEMENT", listAllTasks)

	// Start the server
	port := 8080
	log.Printf("Server is running on port %d\n", port)
	err = router.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
}

func createTask(c *gin.Context) {
	var newTask Task

	if err := c.ShouldBindJSON(&newTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Parse the due_date string to a time.Time value
	dueDate, err := time.Parse("2006-01-02", newTask.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due_date format"})
		return
	}

	// Format the time.Time value back to a string in the desired format
	newTask.DueDate = dueDate.Format("2006-01-02")

	result, err := db.Exec(`
		INSERT INTO MANAGEMENT (title, description, due_date, status)
		VALUES (?, ?, ?, ?)
	`, newTask.Title, newTask.Description, newTask.DueDate, newTask.Status)
	if err != nil {
		log.Println("Error executing SQL query:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	taskID, _ := result.LastInsertId()
	newTask.ID = int(taskID)

	c.JSON(http.StatusCreated, newTask)
}

func retrieveTask(c *gin.Context) {
	taskID := c.Param("id")

	fmt.Println("Attempting to retrieve task with ID:", taskID)

	var task Task
	err := db.QueryRow(`
        SELECT id, title, description, due_date, status FROM MANAGEMENT
        WHERE id = ?
    `, taskID).Scan(&task.ID, &task.Title, &task.Description, &task.DueDate, &task.Status)

	if err != nil {
		fmt.Println("Error retrieving task:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	fmt.Println("Task retrieved successfully:", task)
	c.JSON(http.StatusOK, task)
}

// ...

func updateTask(c *gin.Context) {
	taskID := c.Param("id")

	var updatedTask Task
	if err := c.ShouldBindJSON(&updatedTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Check if the task with the given ID exists
	var existingTask Task
	err := db.QueryRow(`
		SELECT id, title, description, due_date, status FROM MANAGEMENT
		WHERE id = ?
	`, taskID).Scan(&existingTask.ID, &existingTask.Title, &existingTask.Description, &existingTask.DueDate, &existingTask.Status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Perform the update
	_, err = db.Exec(`
		UPDATE MANAGEMENT
		SET title = ?, description = ?, due_date = ?, status = ?
		WHERE id = ?
	`, updatedTask.Title, updatedTask.Description, updatedTask.DueDate, updatedTask.Status, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Retrieve the updated task details
	var updatedTaskDetails Task
	err = db.QueryRow(`
		SELECT id, title, description, due_date, status FROM MANAGEMENT
		WHERE id = ?
	`, taskID).Scan(&updatedTaskDetails.ID, &updatedTaskDetails.Title, &updatedTaskDetails.Description, &updatedTaskDetails.DueDate, &updatedTaskDetails.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated task details"})
		return
	}

	c.JSON(http.StatusOK, updatedTaskDetails)
}

func deleteTask(c *gin.Context) {
	taskID := c.Param("id")

	result, err := db.Exec(`
		DELETE FROM MANAGEMENT
		WHERE id = ?
	`, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func listAllTasks(c *gin.Context) {
	var tasks []Task

	rows, err := db.Query(`
		SELECT id, title, description, due_date, status FROM MANAGEMENT
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.DueDate, &task.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan tasks"})
			return
		}
		tasks = append(tasks, task)
	}

	if len(tasks) > 0 {
		c.JSON(http.StatusOK, tasks)
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "No tasks found"})
	}
}
