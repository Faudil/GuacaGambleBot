import json
from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry

class TutorialQuest(Quest):
    id = "tutorial"
    type = QuestType.MAIN

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.tutorial.step0_welcome",
                rewards=QuestReward(money=100) # Small xp for reading
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.tutorial.step1_activity",
                rewards=QuestReward(money=100),
                target_stat="daily_uses",
                target_count=1
            ),
            QuestStep(
                step_type=QuestStepType.CHOICE,
                text_key="quests.tutorial.step2_choice",
                choices=[
                    {"id": "gambler", "text_ref": "quests.tutorial.choice_gambler"},
                    {"id": "farmer", "text_ref": "quests.tutorial.choice_farmer"}
                ]
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.tutorial.step3_gambler",
                rewards=QuestReward(money=500, achievement_id="quest_rookie")
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.tutorial.step3_farmer",
                rewards=QuestReward(items=["wheat_seed"], xp=200)
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress['step_index']
        if step_idx != 1:
            return False
            
        step = self.steps[step_idx]
        target_stat = step.extra.get('target_stat')
        if activity != target_stat:
            return False
            
        new_val = current_progress.get('progress_value', 0) + amount
        from src.database.quest import update_quest_progress
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= step.extra.get('target_count', 1)

    def on_choice(self, user_id: int, choice_id: str, current_progress: Dict[str, Any]) -> int:
        from src.database.quest import update_quest_progress
        
        # Store permanent choice
        data = current_progress.get('custom_data', {})
        data['tutorial_path'] = choice_id
        update_quest_progress(user_id, self.id, custom_data=data)
        
        if choice_id == "gambler":
            return 3
        else:
            return 4

# Register the quest
QuestRegistry.register(TutorialQuest)
